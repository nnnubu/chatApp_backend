package message

import (
	"ChatApp/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

type MarkReadStatusService struct{}

func NewMarkReadStatusService() *MarkReadStatusService {
	return &MarkReadStatusService{}
}

func (mrs *MarkReadStatusService) MarkReadStatus(ctx context.Context, db *gorm.DB, conversationUid string, currentUid string) error {
	isMember, err := model.IsConversationMember(ctx, db, conversationUid, currentUid)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("无权操作该会话")
	}
	err = model.UpdateConversationRead(ctx, db, conversationUid, currentUid)
	if err != nil {
		return err
	}
	return nil
}
