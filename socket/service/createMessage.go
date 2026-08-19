package service

import (
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/model"
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func CreateMessage(ctx context.Context, db *gorm.DB, senderUid string, chatReq *dto.ChatReq, newMsgId string) error {
	var msg *model.Message
	conversationUid := chatReq.ConversationUID
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取存储中的发起者与接收者的合法会话
		exist, storedConversationUid, err := model.GetPrivateConversation(ctx, tx, senderUid, chatReq.ReceiverUID)
		if err != nil {
			return err
		}
		if !exist {
			// 未发起过私聊会话 却传输了 conversationUid
			if conversationUid != "" {
				global.Log.Error("非法的会话id")
				return err
			}
			/**
			若两者未发起过私聊会话 且未携带会话id 则新设置一个会话id
			不过这种情况是和陌生人私聊 也就是还不是好友的情况下才会触发
			但是我目前还没做和陌生人私聊的入口 仅好友可以私聊 所以此处先占位不做
			**/
			//conversationUid = utils.GenAutoSnowId()
		}

		// 若两者存在过私聊会话 且传输了 conversationUid 则进行判断会话合法与否
		if conversationUid != "" {
			if conversationUid != storedConversationUid {
				global.Log.Error("非法的会话id")
				return err
			}
		} else {
			conversationUid = storedConversationUid
		}

		// 会话合法
		msg = &model.Message{
			MsgID:           newMsgId,
			ReceiverUID:     chatReq.ReceiverUID,
			SenderUID:       senderUid,
			ConversationUID: conversationUid,
			MsgType:         model.MsgTypePrivateChat,
			Content:         chatReq.Content,
		}
		err = model.CreateMessage(ctx, tx, msg)
		if err != nil {
			global.Log.Error(err.Error(), zap.Any("uid", chatReq.ReceiverUID))
			return err
		}
		return nil
	})

	if err != nil {
		global.Log.Error(err.Error())
		return err
	}

	// 消息入库之后 开始推送
	go func(senderUid, receiverUid, conversationUid, msgId, content string) {
		var pushErr error
		taskCtx := context.Background()
		sender, exist, err := model.GetUserByUID(taskCtx, db, senderUid)
		if err != nil || !exist {
			global.Log.Error("推送消息：查询发送用户失败", zap.String("msgId", msgId), zap.Error(err))
			return
		}

		receiver, exist, err := model.GetUserByUID(taskCtx, db, receiverUid)
		if err != nil || !exist {
			global.Log.Error("推送消息：查询接收用户失败", zap.String("msgId", msgId), zap.Error(err))
			return
		}

		// 将接收人的基础信息推送给消息发送者
		pushErr = PushBroadCastMsg(sender.UID, "chat", msgId, dto.ChatResp{
			Uid:             receiver.UID,
			SenderUID:       sender.UID,
			Nickname:        receiver.Nickname,
			AvatarUrl:       receiver.Avatar,
			ConversationUID: conversationUid,
			Content:         content,
		})
		if pushErr != nil {
			return
		}
		// 将发起人的基础信息推送给消息接收者
		pushErr = PushBroadCastMsg(receiver.UID, "chat", msgId, dto.ChatResp{
			Uid:             sender.UID,
			SenderUID:       sender.UID,
			Nickname:        sender.Nickname,
			AvatarUrl:       sender.Avatar,
			ConversationUID: conversationUid,
			Content:         content,
		})
		if pushErr != nil {
			return
		}
	}(senderUid, chatReq.ReceiverUID, conversationUid, msg.MsgID, msg.Content)
	return nil
}
