package friend

import (
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/model"
	"ChatApp/socket/service"
	"ChatApp/utils"
	"context"
	"errors"
	"log"

	"gorm.io/gorm"
)

type UpdateApplyService struct{}

func NewUpdateApplyService() *UpdateApplyService {
	return &UpdateApplyService{}
}

func (uas *UpdateApplyService) UpdateApply(ctx context.Context, db *gorm.DB, ufq *dto.UpdateFriendApplyRequest, currentUid string) error {

	// 判断该好友请求存在与否
	applyRecord, err := model.GetFriendApplyByMsgId(ctx, db, ufq.MsgId)
	if err != nil {
		return errors.New("该好友申请不存在")
	}

	if applyRecord.Status != 0 {
		return errors.New("该好友申请已处理")
	}

	// 判断接收人是否是当前用户
	if applyRecord.TargetUid != currentUid {
		return errors.New("无权操作这条好友申请")
	}

	// 判断是否已经是好友
	isFriend, err := model.IsFriend(ctx, db, currentUid, applyRecord.ApplyUid)
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}
	if isFriend {
		return errors.New("你们已经是好友")
	}

	var status int8
	var lastMsg string
	var lastMsgID string
	var conversationUID string
	if ufq.IsAgree {
		status = 1
		lastMsg = "我已同意你的好友请求，快来聊天吧"
	} else {
		status = 2
		lastMsg = "对方已拒绝你的好友请求"
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 事务内部 用 事务的 context
		txCtx := tx.Statement.Context
		var transactionErr error
		if transactionErr = model.UpdateFriendApplyStatus(txCtx, tx, applyRecord.MsgId, status); transactionErr != nil {
			return transactionErr
		}
		if ufq.IsAgree {
			var exists bool

			if transactionErr = model.CreateFriend(txCtx, tx, currentUid, applyRecord.ApplyUid); transactionErr != nil {
				return transactionErr
			}
			if transactionErr = model.CreateFriend(txCtx, tx, applyRecord.ApplyUid, currentUid); transactionErr != nil {
				return transactionErr
			}

			// 在闭包内部会新建局部变量 遮蔽外部同名变量
			// 此处不要使用 := 否则外部的 conversationUID 会没法得到此处的指
			exists, conversationUID, transactionErr = model.GetPrivateConversation(txCtx, tx, currentUid, applyRecord.ApplyUid)
			if transactionErr != nil {
				return transactionErr
			}
			// 如果 会话id 不存在 则新建一个
			if !exists {
				conversationUID = utils.GenAutoSnowId()
				// 创建会话
				if transactionErr = model.CreateConversation(txCtx, tx, &model.Conversation{
					ConversationUID:  conversationUID,
					ConversationType: model.MsgTypePrivateChat,
				}); transactionErr != nil {
					return transactionErr
				}

				// 创建会话成员
				if transactionErr = model.CreateConversationMember(txCtx, tx, &model.ConversationMember{
					ConversationUID: conversationUID,
					UID:             currentUid,
				}); transactionErr != nil {
					return transactionErr
				}

				if transactionErr = model.CreateConversationMember(txCtx, tx, &model.ConversationMember{
					ConversationUID: conversationUID,
					UID:             applyRecord.ApplyUid,
				}); transactionErr != nil {
					return transactionErr
				}
			}
			// 初始请求消息入库
			if transactionErr = model.CreateMessage(txCtx, tx, &model.Message{
				MsgID:           applyRecord.MsgId,
				ReceiverUID:     currentUid,
				SenderUID:       applyRecord.ApplyUid,
				ConversationUID: conversationUID,
				MsgType:         model.MsgTypePrivateChat,
				Content:         applyRecord.Msg,
			}); transactionErr != nil {
				return transactionErr
			}
			// 请求结果消息入库
			lastMsgID = utils.GenAutoSnowId()
			if transactionErr = model.CreateMessage(txCtx, tx, &model.Message{
				MsgID:           lastMsgID,
				ReceiverUID:     applyRecord.ApplyUid,
				SenderUID:       currentUid,
				ConversationUID: conversationUID,
				MsgType:         model.MsgTypePrivateChat,
				Content:         lastMsg,
			}); transactionErr != nil {
				return transactionErr
			}

			// 同意时 自动标记该好友请求已读 因为该好友请求已经移入消息表独立区分已读状态
			// 否则就会出现 明明已经同意并开始聊天 下次打开却还会拉取未读的好友请求处理结果的情况
			if transactionErr = model.UpdateFriendApplyRead(txCtx, tx, applyRecord.MsgId); transactionErr != nil {
				return transactionErr
			}
		}
		return nil
	})

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	currentUser, isTargetUserExist, err := model.GetUserByUID(ctx, db, currentUid)

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if !isTargetUserExist {
		return errors.New("该用户不存在")
	}

	applyUser, isApplyUserExist, err := model.GetUserByUID(ctx, db, applyRecord.ApplyUid)

	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}

	if !isApplyUserExist {
		return errors.New("该用户不存在")
	}

	curUid := currentUser.UID
	curNick := currentUser.Nickname
	curAvatar := currentUser.Avatar

	appUid := applyUser.UID
	appNick := applyUser.Nickname
	appAvatar := applyUser.Avatar

	applyMsgId := applyRecord.MsgId
	applyTargetUid := applyRecord.TargetUid
	applyApplyUid := applyRecord.ApplyUid
	go func(
		isAgree bool,
		curUid string,
		curNick string,
		curAvatar string,
		appUid string,
		appNick string,
		appAvatar string,
		applyMsgId string,
		applyTargetUid string,
		applyApplyUid string,
		applyMsg string,
		applyMsgID string,
		conversationUID string,
		lastMsgID string,
		lastMsg string,
		status int8) {
		var pushErr error

		// 无论同意与否 都需要推送 addFriend 类消息的结果
		// 因为同意时 前端需要根据该结果实时动态增加到好友列表
		// 失败时 当然也需要知会申请方

		// 将当前接收人的基础信息返回给申请人
		if pushErr = service.PushBroadCastMsg(applyApplyUid, "addFriend", applyMsgId, dto.AddFriendResponse{
			Uid:       curUid,
			ApplyUid:  applyApplyUid,
			TargetUid: curUid,
			Nickname:  curNick,
			Content:   lastMsg,
			AvatarUrl: curAvatar,
			Status:    status,
		}); pushErr != nil {
			global.Log.Error(pushErr.Error())
		}
		// 将申请人的基础信息返回给当前接收人
		if pushErr = service.PushBroadCastMsg(curUid, "addFriend", applyMsgId, dto.AddFriendResponse{
			Uid:       appUid,
			ApplyUid:  applyApplyUid,
			TargetUid: curUid,
			Nickname:  appNick,
			Content:   lastMsg,
			AvatarUrl: appAvatar,
			Status:    status,
		}); pushErr != nil {
			global.Log.Error(pushErr.Error())
		}

		if isAgree {
			// 如果已同意则推送聊天类数据给双方
			log.Println(conversationUID)

			// 注意 推送给 B 用户uid 必须永远是 A 基础信息也得是 A 的
			// 前端 需要根据 用户uid 来渲染消息列表中的卡片

			// 推送第一条消息 发起好友请求的信息
			if pushErr = service.PushBroadCastMsg(appUid, "chat", applyMsgId, dto.ChatResp{
				Uid:             curUid,
				SenderUID:       appUid,
				Nickname:        curNick,
				AvatarUrl:       curAvatar,
				ConversationUID: conversationUID,
				Content:         applyMsg,
			}); pushErr != nil {
				global.Log.Error(pushErr.Error())
			}

			if pushErr = service.PushBroadCastMsg(curUid, "chat", applyMsgId, dto.ChatResp{
				Uid:             appUid,
				SenderUID:       appUid,
				Nickname:        appNick,
				AvatarUrl:       appAvatar,
				ConversationUID: conversationUID,
				Content:         applyMsg,
			}); pushErr != nil {
				global.Log.Error(pushErr.Error())
			}

			// 推送第二条后端自动生成的回执消息
			// 将当前接收人的基础信息推送给申请人
			if pushErr = service.PushBroadCastMsg(appUid, "chat", lastMsgID, dto.ChatResp{
				Uid:             curUid,
				SenderUID:       curUid,
				Nickname:        curNick,
				AvatarUrl:       curAvatar,
				ConversationUID: conversationUID,
				Content:         lastMsg,
			}); pushErr != nil {
				global.Log.Error(pushErr.Error())
			}

			// 将申请人的基础信息推送给当前接收人
			if pushErr = service.PushBroadCastMsg(curUid, "chat", lastMsgID, dto.ChatResp{
				Uid:             appUid,
				SenderUID:       curUid,
				Nickname:        appNick,
				AvatarUrl:       appAvatar,
				ConversationUID: conversationUID,
				Content:         lastMsg,
			}); pushErr != nil {
				global.Log.Error(pushErr.Error())
			}
		}
	}(ufq.IsAgree, curUid, curNick, curAvatar, appUid, appNick, appAvatar, applyMsgId, applyTargetUid, applyApplyUid, applyRecord.Msg, applyRecord.MsgId, conversationUID, lastMsgID, lastMsg, status)

	return nil
}
