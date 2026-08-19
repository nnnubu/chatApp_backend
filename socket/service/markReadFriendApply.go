package service

import (
	"ChatApp/model"
	"context"
	"errors"

	"gorm.io/gorm"
)

func MarkFriendApplyRead(ctx context.Context, db *gorm.DB, uid, msgId string) error {
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 根据msgId查出这条好友申请记录
		apply, err := model.GetFriendApplyByMsgId(ctx, tx, msgId)
		if err != nil {
			// 记录不存在：msgId 错误 或 已被删除
			return errors.New("好友申请记录不存在")
		}
		if apply.ApplyUid != uid {
			return errors.New("无权限操作该好友申请")
		}

		if err = model.UpdateFriendApplyRead(ctx, tx, msgId); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return errors.New("系统繁忙，请稍后重试")
	}
	return nil
}
