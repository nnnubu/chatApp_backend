package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UserSticker 用户收藏的自定义表情包
// 用户上传图片作为表情包收藏，发送时直接引用 url，无需每次上传
type UserSticker struct {
	ID        string    `gorm:"size:36;primaryKey;comment:表情包id" json:"id"`
	UID       string    `gorm:"size:36;not null;index:idx_uid;comment:所属用户uid" json:"uid"`
	URL       string    `gorm:"size:255;not null;comment:表情包图片url" json:"url"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// CreateUserSticker 创建用户表情包收藏记录
func CreateUserSticker(ctx context.Context, db *gorm.DB, sticker *UserSticker) error {
	return db.WithContext(ctx).Create(sticker).Error
}

// GetUserStickersByUID 拉取用户收藏的表情包列表（按创建时间倒序）
func GetUserStickersByUID(ctx context.Context, db *gorm.DB, uid string) ([]UserSticker, error) {
	var stickers []UserSticker
	err := db.WithContext(ctx).
		Where("uid = ?", uid).
		Order("created_at DESC").
		Find(&stickers).Error
	return stickers, err
}

// DeleteUserSticker 删除用户收藏的表情包（仅能删除自己的）
// 返回受影响行数，0 表示不存在或无权删除
func DeleteUserSticker(ctx context.Context, db *gorm.DB, id string, uid string) (int64, error) {
	result := db.WithContext(ctx).
		Where("id = ? AND uid = ?", id, uid).
		Delete(&UserSticker{})
	return result.RowsAffected, result.Error
}
