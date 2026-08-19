package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// gorm给字段自动命名的规则是根据这个驼峰命名法来设计的，大写字母就直接用_隔离了
// constraint用于设置级联操作 OnUpdate是主表的关联字段值更新时同步更新从表的关联字段值 OnDelete也是同样原理
// OnDelete:CASCADE: 主表记录删除时，自动删除从表记录（如果是原生的数据库删除指令，需要先删除从表记录，才能删除主表记录）
// OnDelete:SET NULL：主表记录删除时，从表外键字段设为 NULL（需保证外键字段允许为 NULL）。
// OnDelete:RESTRICT：如果从表存在关联记录，主表记录不允许删除（默认行为）。

type FriendApply struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	MsgId     string    `gorm:"size:40;unique;not null;index"`
	ApplyUid  string    `gorm:"size:36;not null;index:idx_apply_target,priority:1"`
	TargetUid string    `gorm:"size:36;not null;index:idx_apply_target,priority:2"`
	Msg       string    `gorm:"size:255;" json:"msg"`
	Status    int8      `gorm:"type:tinyint;default:0;comment:验证状态(0:待处理,1:同意,2:拒绝,3:过期)" json:"status"`
	ApplyRead int8      `gorm:"type:tinyint;default:0;comment:申请人是否已读处理结果 0:未读,1:已读" json:"applyRead"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// uniqueIndex:uk_apply_pair,priority:1 和 uniqueIndex:uk_apply_pair,priority:1 相当于
// create unique index uk_apply_pair on friend_apply (apply_uid, target_uid);

type Friend struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"-"`
	Uid       string    `gorm:"size:36;not null;uniqueIndex:uk_user_friend_pair,priority:1" json:"uid"`
	FriendUid string    `gorm:"size:36;not null;uniqueIndex:uk_user_friend_pair,priority:2" json:"friendUid"`
	Remark    string    `gorm:"size:50;default:'';comment:好友备注" json:"remark"`
	Status    int8      `gorm:"type:tinyint;default:0;comment:好友状态(0:正常,1:拉黑)" json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// create unique index uk_user_friend_pair on friend (uid,friend_uid)

// CreateFriendApply 发起好友请求
func CreateFriendApply(ctx context.Context, txDB *gorm.DB, apply *FriendApply) error {
	if err := txDB.WithContext(ctx).Create(apply).Error; err != nil {
		return err
	}
	return nil
}

// GetFriendApplyByMsgId 获取好友请求
func GetFriendApplyByMsgId(ctx context.Context, db *gorm.DB, msgId string) (*FriendApply, error) {
	var apply = &FriendApply{}
	err := db.WithContext(ctx).Model(&FriendApply{}).Where("msg_id = ?", msgId).First(&apply).Error
	if err != nil {
		return nil, err
	}
	return apply, nil
}

// FriendApplyHasPending 是否重复发起好友请求
func FriendApplyHasPending(ctx context.Context, db *gorm.DB, currentUid, targetUid string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&FriendApply{}).Where("apply_uid = ? and target_uid = ? and status = 0", currentUid, targetUid).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// IsTargetPending 对方是否已向自己发送好友请求
func IsTargetPending(ctx context.Context, db *gorm.DB, currentUid, targetUid string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&FriendApply{}).Where("apply_uid = ? and target_uid = ? and status = 0", targetUid, currentUid).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateFriendApplyStatus 更新好友请求状态
func UpdateFriendApplyStatus(ctx context.Context, txDB *gorm.DB, msgId string, status int8) error {
	if err := txDB.WithContext(ctx).Model(&FriendApply{}).Where("msg_id = ? and status = 0", msgId).Update("status", status).Error; err != nil {
		return err
	}
	return nil
}

// UpdateFriendApplyRead 更新好友请求申请方为已读
func UpdateFriendApplyRead(ctx context.Context, txDB *gorm.DB, msgId string) error {
	if err := txDB.WithContext(ctx).Model(&FriendApply{}).Where("msg_id = ? and apply_read = 0", msgId).Update("apply_read", 1).Error; err != nil {
		return err
	}
	return nil
}

// GetPendingApplyByUid 通过 uid 获取用户所有未处理的好友请求
func GetPendingApplyByUid(ctx context.Context, db *gorm.DB, currentUid string) ([]FriendApply, error) {
	var list []FriendApply
	err := db.WithContext(ctx).Model(&FriendApply{}).Where("target_uid = ? and status = 0", currentUid).Find(&list).Error
	if err != nil {

		return nil, err
	}
	return list, nil
}

// GetUnReadApplyByUid 通过 uid 获取用户发起并且已被处理但是自身未读的好友请求
func GetUnReadApplyByUid(ctx context.Context, db *gorm.DB, currentUid string) ([]FriendApply, error) {
	var list []FriendApply
	err := db.WithContext(ctx).Where("apply_uid = ? and status in (1,2) and apply_read = 0", currentUid).Find(&list).Error
	if err != nil {

		return nil, err
	}
	return list, err
}

// IsFriend 是否已经是好友
func IsFriend(ctx context.Context, db *gorm.DB, currentUid, targetUid string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&Friend{}).Where("((uid = ? and friend_uid = ?) or (uid = ? and friend_uid = ?)) and status = 0",
		currentUid, targetUid, targetUid, currentUid).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateFriend 创建好友关系
func CreateFriend(ctx context.Context, txDB *gorm.DB, currentUid, targetUid string) error {
	return txDB.WithContext(ctx).Create(&Friend{
		Uid:       currentUid,
		FriendUid: targetUid,
	}).Error
}

func PullFriendsByUid(ctx context.Context, db *gorm.DB, uid string, page int, pageSize int) ([]Friend, bool, error) {
	// *gorm.DB 是可变引用对象 链式调用修改内部 Statement 结构体 部分操作会修改原对象 部分会克隆副本 不同行为结果不一样
	// query 要记得分开写 或者 使用 Session 克隆副本 如果共用一个 query 比如说先 使用 query.Offset(offset).Limit(pageSize).Order("id desc")
	// 然后使用 query.Count 就会把前面的查询条件清空 后面再使用 find 就会失去查询条件而出现错误
	// Count 内部会克隆副本统计，原对象保留 Where 但是对于 offset limit join preload select distinct 就会清空
	//var total int64
	var list []Friend
	// list 初始是 nil 如果直接返回 空切片 与 nil == 判断是 true
	// 但是 gorm 的 find 内部会执行 append 或者重置切片 自动初始化为 []Friend{} 此时就不是 nil 了

	//countQuery := db.WithContext(ctx).Model(&Friend{}).Where("uid = ?", uid)
	//// 获取好友总数
	//if err := countQuery.Count(&total).Error; err != nil {
	//	return nil, 0, err
	//}

	// 偏移量 即 起始好友位置
	offset := (page - 1) * pageSize
	pageQuery := db.WithContext(ctx).Model(&Friend{}).Where("uid = ?", uid)
	// 从偏移寻找 限制长度为 pageSize
	err := pageQuery.Offset(offset).Limit(pageSize + 1).Order("id desc").Find(&list).Error
	if err != nil {
		return nil, false, err
	}
	hasMore := false
	if len(list) > pageSize {
		list = list[:pageSize]
		hasMore = true
	}
	return list, hasMore, nil
}
