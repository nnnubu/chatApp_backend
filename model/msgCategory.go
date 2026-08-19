package model

import (
	"ChatApp/utils"
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	CategoryTypeFriend = 1 // 好友
	CategoryTypeGroup  = 2 // 群聊
	CategoryTypeCustom = 3 // 自定义分类
)

type MsgCategory struct {
	ID           string    `gorm:"primary_key;size:36;comment:雪花id" json:"-"`
	UID          string    `gorm:"size:36;not null;uniqueIndex:idx_uid_name;comment:uuid v7用户唯一标识" json:"uid"`
	CategoryName string    `gorm:"size:32;not null;uniqueIndex:idx_uid_name;comment:分类名称" json:"cateGoryName"`
	CategoryType int8      `gorm:"type:tinyint;not null;default:3;comment:(1:系统内置好友,2:系统内置群聊,3:用户自定义分类)" json:"cateGoryType"`
	Sort         int       `gorm:"default:0;comment:排序权重" json:"sort"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// InitUserDefaultCategory 初始化用户内置分类
func InitUserDefaultCategory(ctx context.Context, tx *gorm.DB, uid string) error {
	list := []MsgCategory{
		{ID: utils.GenAutoSnowId(), UID: uid, CategoryType: CategoryTypeFriend, CategoryName: "好友"},
		{ID: utils.GenAutoSnowId(), UID: uid, CategoryType: CategoryTypeGroup, CategoryName: "群聊"},
	}
	return tx.WithContext(ctx).Create(&list).Error
}

func GetCategoryByUID(ctx context.Context, db *gorm.DB, uid string) ([]MsgCategory, error) {
	var list []MsgCategory
	err := db.WithContext(ctx).Model(&MsgCategory{}).Where("uid = ?", uid).Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
