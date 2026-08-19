package model

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const (
	MsgTypePrivateChat  = 1 // 私聊消息
	MsgTypeGroupChat    = 2 // 群聊消息
	MsgTypeSystemNotice = 3 // 系统通知
)

// Message 消息表
type Message struct {
	MsgID           string    `gorm:"size:36;primaryKey;comment:雪花id" json:"msgID"`
	ReceiverUID     string    `gorm:"size:36;not null;comment:消息接收用户UID" json:"receiverUID"`
	SenderUID       string    `gorm:"size:36;not null;comment:消息发送用户UID" json:"senderUID"`
	ConversationUID string    `gorm:"size:36;not null;comment:会话UID" json:"conversationUID"`
	MsgType         int8      `gorm:"type:tinyint;not null;comment:消息业务类型" json:"msgType"`
	Content         string    `gorm:"type:text;comment:消息内容" json:"content"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// MessageRead 消息已读表
type MessageRead struct {
	// 两个 primaryKey 联合主键 在删除和更新时 一定要带上两个字段 否则可能会多更新或是多删除
	MsgID  string    `gorm:"size:36;primaryKey;comment:消息id" json:"msgID"`
	UID    string    `gorm:"size:36;primaryKey;comment:哪个用户标记已读" json:"uid"`
	ReadAt time.Time `gorm:"autoCreateTime;comment:已读时间 记录不存在时代表未读" json:"readAt"`
}

// Conversation 会话表
type Conversation struct {
	ConversationUID  string    `gorm:"size:36;primaryKey;comment:会话UID" json:"conversationUID"`
	ConversationType int8      `gorm:"type:tinyint;not null;comment:1私聊,2群聊" json:"conversationType"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// ConversationMember 会话成员表
type ConversationMember struct {
	ConversationUID string `gorm:"size:36;primaryKey"`
	UID             string `gorm:"size:36;primaryKey;comment:归属于该会话的用户UID" json:"uid"`
}

// CreateMessage 消息入库
func CreateMessage(ctx context.Context, db *gorm.DB, msg *Message) error {
	err := db.WithContext(ctx).Model(&Message{}).Create(msg).Error
	if err != nil {
		return err
	}
	return nil
}

// GetPrivateConversationByUID 获取私聊会话 同时也可判断会话是不是私聊类型 当前访问成员是否属于该会话
func GetPrivateConversationByUID(ctx context.Context, db *gorm.DB, currentUid string, conversationUID string) (*Conversation, error) {
	// 找出指定私聊会话
	var conversation Conversation
	err := db.WithContext(ctx).Model(&Conversation{}).
		Table("conversations as c").
		Joins("inner join conversation_members as cm on cm.conversation_uid = c.conversation_uid").
		Where("c.conversation_uid = ?", conversationUID).
		Where("c.conversation_type = ?", MsgTypePrivateChat).
		Where("cm.uid = ?", currentUid).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// GetConversationMember 获取会话成员
func GetConversationMember(ctx context.Context, db *gorm.DB, currentUid string, conversationUID string) (uidList []string, err error) {
	err = db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversation_uid = ? and uid <> ?", conversationUID, currentUid).Select("uid").Find(&uidList).Error
	if err != nil {
		return nil, err
	}
	return uidList, nil
}

// GetPrivateConversation 获取用户的私聊会话
func GetPrivateConversation(ctx context.Context, db *gorm.DB, currentUID string, targetUID string) (exists bool, conversationUID string, err error) {
	// 找出当前用户所有的 会话id
	subQuery := db.WithContext(ctx).Model(&ConversationMember{}).Select("conversation_uid").Where("uid = ?", currentUID)

	var res struct {
		ConversationUID string
	}

	/**
		联表查询 ConversationMember表 和 Conversation表 中同一会话id 的内容
		即 建立会话id 和 用户id 的映射 类似 key 为 会话id value 为 用户id集合 的关系
		条件 cm.uid = targetUID 找出 目标用户的所有会话id
		cm.conversation_uid in subQuery  当前用户是否与目标用户有共同的会话id
		c.conversation_type = MsgTypePrivateChat 共同的会话id中类型仅为私聊的
		不能联表两个都是 ConversationMember 否则 无法区分是 私聊消息 还是群聊消息
	**/
	err = db.WithContext(ctx).Model(&ConversationMember{}).
		Table("conversation_members as cm").
		Select("cm.conversation_uid").
		Joins("inner join conversations as c on cm.conversation_uid = c.conversation_uid").
		Where("cm.uid = ?", targetUID).
		Where("cm.conversation_uid in (?)", subQuery).
		Where("c.conversation_type = ?", MsgTypePrivateChat).Limit(1).Scan(&res).Error

	if err != nil {
		return false, "", err
	}
	if res.ConversationUID != "" {
		return true, res.ConversationUID, nil
	}
	return false, "", nil
}

// CreateConversation 创建会话
func CreateConversation(ctx context.Context, db *gorm.DB, conversation *Conversation) error {
	err := db.WithContext(ctx).Create(conversation).Error
	if err != nil {
		return err
	}
	return nil
}

// CreateConversationMember 创建会话成员
func CreateConversationMember(ctx context.Context, db *gorm.DB, conversationMember *ConversationMember) error {
	err := db.WithContext(ctx).Create(conversationMember).Error
	if err != nil {
		return err
	}
	return nil
}

// PullHistoryMessage 游标拉取历史消息
func PullHistoryMessage(ctx context.Context, db *gorm.DB, pageSize int, cursorMsgId string, conversationUid string) (list []*Message, hasMore bool, err error) {
	// 消息表中 滤出指定会话id 的所有内容
	query := db.WithContext(ctx).Model(&Message{}).Where("conversation_uid = ?", conversationUid)
	// 游标不为空 找出所有 msgId < 当前值的消息 因为 雪花 id 大 = 新消息 小 = 旧消息
	if cursorMsgId != "" {
		query = query.Where("msg_id < ?", cursorMsgId)
	}
	// desc 索引从新消息往旧消息排	若游标为空 则自动拉取最新的 pageSize 条消息
	err = query.Order("msg_id desc").Limit(pageSize + 1).Find(&list).Error
	if err != nil {
		return nil, false, err
	}
	hasMore = false
	if len(list) > pageSize {
		list = list[:pageSize]
		hasMore = true
	}
	return list, hasMore, nil
}
