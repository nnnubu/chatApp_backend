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
	// 使用伪字段来专门存放索引标签 下划线不会生成数据库列
	_ struct{} `gorm:"index:idx_conversation_msgId,columns:conversation_uid,msg_id"`
}

// MessageRead 消息已读表
//type MessageRead struct {
//	// 两个 primaryKey 联合主键 在删除和更新时 一定要带上两个字段 否则可能会多更新或是多删除
//	MsgID  string    `gorm:"size:36;primaryKey;comment:消息id" json:"msgID"`
//	UID    string    `gorm:"size:36;primaryKey;comment:哪个用户标记已读" json:"uid"`
//	ReadAt time.Time `gorm:"autoCreateTime;comment:已读时间 记录不存在时代表未读" json:"readAt"`
//}

// ConversationRead 不再使用上面这个表作为已读 否则更新已读需要插入大量数据 采用如下方式来更新已读则只需要插入一次数据更新一条字段
type ConversationRead struct {
	// 联合主键已经约束了 同一个用户 + 同一个会话 不会出现多条记录
	UID             string    `gorm:"size:36;primaryKey" json:"uid"`
	ConversationUID string    `gorm:"size:36;primaryKey" json:"conversationUID"`
	LastReadMsgID   string    `gorm:"size:36;not null;comment:用户在该会话读到的最后一条消息msgId" json:"LastReadMsgID"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// Conversation 会话表
type Conversation struct {
	ConversationUID  string    `gorm:"size:36;primaryKey;comment:会话UID" json:"conversationUID"`
	ConversationType int8      `gorm:"type:tinyint;not null;comment:1私聊,2群聊" json:"conversationType"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// ConversationMember 会话成员表
type ConversationMember struct {
	ConversationUID string `gorm:"size:36;primaryKey" json:"conversationUID"`
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
func GetConversationMember(ctx context.Context, db *gorm.DB, conversationUID string) (uidList []string, err error) {
	err = db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversation_uid = ?", conversationUID).Select("uid").Find(&uidList).Error
	if err != nil {
		return nil, err
	}
	return uidList, nil
}

// GetPrivateConversation 获取用户的私聊会话
func GetPrivateConversation(ctx context.Context, db *gorm.DB, currentUID string, targetUID string) (exists bool, conversationUID string, err error) {
	var res struct {
		ConversationUID string
	}

	// 找出当前用户所有的 会话id
	//subQuery := db.WithContext(ctx).Model(&ConversationMember{}).Select("conversation_uid").Where("uid = ?", currentUID)

	/**
		联表查询 ConversationMember表 和 Conversation表 中同一会话id 的内容
		即 建立会话id 和 用户id 的映射 类似 key 为 会话id value 为 用户id集合 的关系
		条件 cm.uid = targetUID 找出 目标用户的所有会话id
		cm.conversation_uid in subQuery  当前用户是否与目标用户有共同的会话id
		c.conversation_type = MsgTypePrivateChat 共同的会话id中类型仅为私聊的
		不能联表两个都是 ConversationMember 否则 无法区分是 私聊消息 还是群聊消息
	**/
	//err = db.WithContext(ctx).Model(&ConversationMember{}).
	//	Table("conversation_members as cm").
	//	Select("cm.conversation_uid").
	//	Joins("inner join conversations as c on cm.conversation_uid = c.conversation_uid").
	//	Where("cm.uid = ?", targetUID).
	//	Where("cm.conversation_uid in (?)", subQuery).
	//	Where("c.conversation_type = ?", MsgTypePrivateChat).Limit(1).Scan(&res).Error

	// 弃用以上方案 改用下面方案更合适
	// 子查询 找出类型为私聊 并且成员同时包含两个用户的会话
	/*
		Where("uid in (?,?)", currentUID, targetUID) 找出成员是 currentUID 或成员为 targetUID 的记录
		此时结果集 为 currentUID 参与过的会话 targetUID 参与过的会话的 member 记录
		Group("conversation_uid") 按照会话id 分组 同一个会话id 的 uid 会归到同一组中
		Having 与 Where 的区别在于 后者是在分组之前过滤数据用 而 前者是分组之后过滤数据用
		Having("count(DISTINCT uid) = ?", 2) 找出该组中不重复的 uid 数量为 2 的会话
		但是若群聊类型的会话且成员刚好只有两个成员时的记录同样也会命中 因此需要后面再次过滤
	*/
	sub := db.WithContext(ctx).Model(&ConversationMember{}).
		Select("conversation_uid").
		Where("uid in (?,?)", currentUID, targetUID).
		Group("conversation_uid").
		Having("count(DISTINCT uid) = ?", 2)

	err = db.WithContext(ctx).Model(&Conversation{}).
		Select("conversation_uid").
		Where("conversation_type = ?", MsgTypePrivateChat).
		Where("conversation_uid in (?)", sub).
		Limit(1).Scan(&res).Error

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

// PullUnReadMessage 获取用户所有会话的未读消息记录
func PullUnReadMessage(ctx context.Context, db *gorm.DB, currentUid string) ([]Message, error) {
	// 获取用户所有会话id
	subConv := db.Model(&ConversationMember{}).Select("conversation_uid").Where("uid = ?", currentUid)
	var result []Message
	err := db.WithContext(ctx).Model(&Message{}).
		Table("messages as m").
		Joins("join conversation_members as cm on m.conversation_uid = cm.conversation_uid"). // 滤出两个表中会话id相同的内容并进行拼接 即 key 为 会话id value 为 消息表和消息成员表的拼接内容 若某一条会话有 n 个成员 则一条消息拼接出来就会扩成 n 行
		// 结果集：每一条消息 拼接上这条消息所属会话的某一条成员记录
		Joins("left join conversation_reads cr on cr.conversation_uid = m.conversation_uid and cr.uid = cm.uid"). // left join 与普通 join 即 inner join 不同的是后者是拼接两个表中都有的内容 而前者是返回 左表中的所有记录 即使游标中没有匹配的记录 没有匹配的记录会以 null 表示
		// 结果集变化：每一行追加 conversation_read 的字段 有记录就填值 没有记录全部 NULL
		Where("cm.uid = ?", currentUid).
		// 仅保留指定会话成员的记录
		Where("m.conversation_uid in (?)", subConv).
		// 会话id 包含在子查询结果集里 的记录 其实上一步已经差不多拿到了
		Where("cr.last_read_msg_id is null or m.msg_id > cr.last_read_msg_id").
		// 该用户没有已读 或 消息表msg_id比已读表的msg_id大 的消息记录
		Find(&result).Error
	return result, err
}

// UpdateConversationRead 更新用户在某个会话的已读位置
func UpdateConversationRead(ctx context.Context, db *gorm.DB, conversationUid string, currentUid string) error {
	var maxMsgId *string
	err := db.WithContext(ctx).Model(&Message{}).
		Select("max(msg_id)").
		Where("conversation_uid = ?", conversationUid).
		Scan(&maxMsgId).Error
	if err != nil {
		return err
	}
	// 若当前会话无消息 则不更新
	if maxMsgId == nil {
		return nil
	}

	cr := ConversationRead{
		UID:             currentUid,
		ConversationUID: conversationUid,
	}
	// 查询记录存在与否 否则创建新记录 不能无记录更新
	err = db.WithContext(ctx).FirstOrCreate(&cr).Error
	if err != nil {
		return err
	}

	// 若最新消息小于已记录位置则不更新 虽然感觉不是很用得上
	if cr.LastReadMsgID >= *maxMsgId {
		return nil
	}

	// 先保证记录存在再更新
	err = db.WithContext(ctx).Model(&cr).
		Update("last_read_msg_id", *maxMsgId).Error
	if err != nil {
		return err
	}
	return nil
}

// IsConversationMember 校验是否是指定会话中的成员
func IsConversationMember(ctx context.Context, db *gorm.DB, conversationUid string, currentUid string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversation_uid = ? and uid = ?", conversationUid, currentUid).
		Limit(1).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
