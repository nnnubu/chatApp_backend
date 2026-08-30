package service

import (
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/model"
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 消息幂等性检查的 Redis key 前缀
const msgIdempotentPrefix = "msg:idempotent:"

// 幂等性检查的过期时间（5分钟）
const msgIdempotentTTL = 5 * time.Minute

func CreateMessage(ctx context.Context, db *gorm.DB, rc *redis.Client, senderUid string, chatReq *dto.ChatReq, newMsgId string) error {
	var msg *model.Message
	conversationUid := chatReq.ConversationUID

	// 幂等性检查：如果 requestId 已经处理过，直接返回成功，避免重复入库
	if chatReq.RequestId != "" && rc != nil {
		idempotentKey := msgIdempotentPrefix + chatReq.RequestId
		// 尝试设置 key，如果 key 已存在则返回 false
		set, err := rc.SetNX(ctx, idempotentKey, newMsgId, msgIdempotentTTL).Result()
		if err != nil {
			global.Log.Error("幂等性检查失败", zap.String("requestId", chatReq.RequestId), zap.Error(err))
			// Redis 异常时不阻断消息发送，仅记录日志
		} else if !set {
			// key 已存在，说明该 requestId 已经处理过，直接返回成功
			global.Log.Info("消息重复发送，已通过幂等性检查拦截",
				zap.String("requestId", chatReq.RequestId),
				zap.String("msgId", newMsgId))
			return nil
		}
	}

	global.Log.Info("CreateMessage", zap.String("senderUid", senderUid), zap.String("receiverUid", chatReq.ReceiverUID), zap.String("conversationUid", conversationUid))
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取存储中的发起者与接收者的合法会话
		exist, storedConversationUid, err := model.GetPrivateConversation(ctx, tx, senderUid, chatReq.ReceiverUID)
		if err != nil {
			global.Log.Error("GetPrivateConversation error", zap.Error(err))
			return err
		}
		global.Log.Info("GetPrivateConversation result", zap.Bool("exist", exist), zap.String("storedConversationUid", storedConversationUid))
		if !exist {
			// 未发起过私聊会话 却传输了 conversationUid
			if conversationUid != "" {
				global.Log.Error("非法的会话id")
				return errors.New("非法的会话id")
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
				return errors.New("非法的会话id")
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
		// 消息入库失败时，删除幂等性 key，允许前端重试
		if chatReq.RequestId != "" && rc != nil {
			idempotentKey := msgIdempotentPrefix + chatReq.RequestId
			_ = rc.Del(ctx, idempotentKey).Err()
		}
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
			Uid:             sender.UID,
			SenderUID:       sender.UID,
			ReceiverUID:     receiver.UID,
			Nickname:        sender.Nickname,
			AvatarUrl:       sender.Avatar,
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
			ReceiverUID:     receiver.UID,
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
