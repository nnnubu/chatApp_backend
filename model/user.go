package model

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// 自增 ID ：库内联表，数据库外键，内部查询等操作用，不对外展示
// UID ： uuid v7生成的唯一永久不变的字段，用于前端接口路由参数，鉴权，Redis缓存等操作

type User struct {
	ID  uint   `gorm:"primaryKey;autoIncrement" json:"-"`
	UID string `gorm:"size:36;not null;unique;comment:uuid v7用户唯一标识" json:"uid"`
	//Account   string    `gorm:"size:50;unique;not null;comment:用户自定义账号，用于登录、加好友" json:"-"`
	Nickname  string    `gorm:"size:50;not null" json:"nickname"`
	Password  string    `gorm:"size:100;not null" json:"-"` // json为- 意思是密码不返回给前端
	Gender    int8      `gorm:"type:tinyint;default:0;comment:性别(0:保密,1:女,2:男)" json:"gender"`
	Email     string    `gorm:"size:100;not null;unique" json:"email"`
	Avatar    string    `gorm:"size:255;default:'';comment:个人头像" json:"avatar"`
	BgImg     string    `gorm:"size:255;default:'';comment:个人主页背景图" json:"bgImg"`
	Intro     string    `gorm:"size:128;default:'';comment:用户个性简介/签名" json:"intro"`
	Birthday  time.Time `gorm:"type:date;default:null;comment:用户生日，仅存年月日" json:"birthday"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func CreateUser(ctx context.Context, txDB *gorm.DB, user *User) error {
	return txDB.WithContext(ctx).Create(user).Error
}

func IsEmailExists(ctx context.Context, db *gorm.DB, email string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func GetUserByEmail(ctx context.Context, db *gorm.DB, email string) (*User, bool, error) {
	var user User
	err := db.WithContext(ctx).Model(&User{}).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &user, true, nil
}

func GetUserByUID(ctx context.Context, db *gorm.DB, uid string) (*User, bool, error) {
	var user User
	err := db.WithContext(ctx).Model(&User{}).Where("uid = ?", uid).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &user, true, nil
}

func GetUserByList(ctx context.Context, db *gorm.DB, uidList []string) ([]User, error) {
	var userList []User
	err := db.WithContext(ctx).Model(&User{}).Where("uid in ?", uidList).Find(&userList).Error
	if err != nil {
		return nil, err
	}
	return userList, nil
}

func UpdatePassword(ctx context.Context, txDB *gorm.DB, email string, encryptPwd string) error {
	return txDB.WithContext(ctx).Model(&User{}).Where("email = ?", email).Update("password", encryptPwd).Error
}

func UpdateInfo(ctx context.Context, txDB *gorm.DB, updateMap map[string]any, uid string) error {
	return txDB.WithContext(ctx).
		Model(&User{}).
		Where("uid = ?", uid).
		Updates(updateMap).Error
}

func UpdateAvatarUrl(ctx context.Context, txDb *gorm.DB, thumbUrl string, uid string) error {
	// UpdateColumn 与 Update 的区别在于 后者会自动刷新 UpdateAt 而前者不刷新	因为只是资源替换，所以没必要刷新
	return txDb.WithContext(ctx).Model(&User{}).Where("uid = ?", uid).UpdateColumn("avatar", thumbUrl).Error
}

func UpdateBgImgUrl(ctx context.Context, txDb *gorm.DB, thumbUrl string, uid string) error {
	return txDb.WithContext(ctx).Model(&User{}).Where("uid = ?", uid).UpdateColumn("bgImg", thumbUrl).Error
}

// SearchUsers 搜索用户（按昵称或邮箱模糊匹配，排除自己）
func SearchUsers(ctx context.Context, db *gorm.DB, keyword string, excludeUid string, limit int) ([]User, error) {
	var users []User
	err := db.WithContext(ctx).Model(&User{}).
		Where("uid != ?", excludeUid).
		Where("nickname LIKE ? OR email LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(limit).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
