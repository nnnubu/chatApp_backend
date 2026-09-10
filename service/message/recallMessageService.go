package message

import (
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/model"
	"ChatApp/socket/service"
	"ChatApp/utils"
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RecallMessageService struct{}

func NewRecallMessageService() *RecallMessageService {
	return &RecallMessageService{}
}

// RecallMessage 撤回消息：仅发送者可撤回 更新 recalled 标记 并广播撤回事件给会话双方
func (rms *RecallMessageService) RecallMessage(ctx context.Context, db *gorm.DB, rc *redis.Client, senderUid string, req *dto.RecallMessageReq) error {
	msg, exists, err := model.GetMessageByMsgID(ctx, db, req.MsgId)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("消息不存在或已被删除")
	}
	// 只能撤回自己发送的消息
	if msg.SenderUID != senderUid {
		return errors.New("只能撤回自己发送的消息")
	}
	// 已撤回的消息不能重复撤回
	if msg.Recalled {
		return errors.New("消息已撤回，请勿重复操作")
	}
	// 校验当前用户属于该会话
	isMember, err := model.IsConversationMember(ctx, db, msg.ConversationUID, senderUid)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("无权操作该会话")
	}

	// 更新撤回标记
	if err := model.RecallMessage(ctx, db, req.MsgId); err != nil {
		global.Log.Error("撤回消息更新失败", zap.String("msgId", req.MsgId), zap.Error(err))
		return err
	}

	// 广播撤回事件给会话双方（在线实时更新 离线下次拉历史时 recalled=true 天然生效）
	global.Log.Info("消息撤回成功", zap.String("msgId", req.MsgId), zap.String("conversationUid", req.ConversationUID), zap.String("senderUid", senderUid))
	go func() {
		payload := dto.RecallResp{
			MsgId:           msg.MsgID,
			ConversationUID: msg.ConversationUID,
			SenderUID:       msg.SenderUID,
			ReceiverUID:     msg.ReceiverUID,
		}
		recallMsgId := utils.GenAutoSnowId()
		if err := service.PushBroadCastMsg(msg.SenderUID, "recall", recallMsgId, payload); err != nil {
			global.Log.Warn("撤回广播：推送发送者失败（可能不在线）", zap.String("msgId", req.MsgId), zap.Error(err))
		}
		if err := service.PushBroadCastMsg(msg.ReceiverUID, "recall", recallMsgId, payload); err != nil {
			global.Log.Warn("撤回广播：推送接收者失败（可能不在线）", zap.String("msgId", req.MsgId), zap.Error(err))
		}
	}()
	return nil
}
