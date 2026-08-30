package dto

type ImageResp struct {
	Url    string `json:"url"`
	ThumbW int    `json:"thumbW"`
	ThumbH int    `json:"thumbH"`
}

type LoginRequest struct {
	// json 请求体 body 中获取数据
	Email    string `json:"Email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token    string    `json:"token"`
	Uid      string    `json:"uid"`
	NickName string    `json:"nickname"`
	Avatar   ImageResp `json:"avatar"`
	BgImg    ImageResp `json:"bgImg"`
	Gender   int8      `json:"gender"`
	Intro    string    `json:"intro"`
	Birthday string    `json:"birthday"`
}

type RegisterRequest struct {
	Nickname string `json:"nickname" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

type RestPwdRequest struct {
	Email    string `json:"email" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateInfoRequest 字段改成 指针变量 可局部更新参数 当字段未传入时为 nil 此时不对其进行更新
type UpdateInfoRequest struct {
	Nickname *string `json:"nickname"`
	Intro    *string `json:"intro"`
	Gender   *int8   `json:"gender"`
	Birthday *string `json:"birthday"`
}

type GetOthersResponse struct {
	Uid             string    `json:"uid" binding:"required"`
	Nickname        string    `json:"nickname" binding:"required"`
	Intro           string    `json:"intro" binding:"required"`
	Avatar          ImageResp `json:"avatar" binding:"required"`
	BgImg           ImageResp `json:"bgImg" binding:"required"`
	IsFriend        bool      `json:"isFriend" binding:"required"`
	ConversationUid string    `json:"conversationUid" binding:"required"`
}

type AddFriendRequest struct {
	TargetUid string `json:"targetUid" binding:"required"`
	Msg       string `json:"msg" binding:"required"`
}

type AddFriendResponse struct {
	Uid       string `json:"uid" binding:"required"` // 当前卡片部分信息的归属Uid
	ApplyUid  string `json:"applyUid" binding:"required"`
	TargetUid string `json:"targetUid" binding:"required"`
	Nickname  string `json:"nickname" binding:"required"`
	Content   string `json:"content" binding:"required"`
	// 头像就不用返回压缩尺寸了，不然容易模糊
	AvatarUrl string `json:"avatarUrl" binding:"required"`
	Status    int8   `json:"status,omitempty"`
}

type UpdateFriendApplyRequest struct {
	MsgId   string `json:"msgId" binding:"required"`
	IsAgree bool   `json:"isAgree"`
	// IsAgree   bool   `json:"isAgree" binding:"required"`
	// 注意 这个 bool 如果 required 前端传输 false 会被自动判定为 required 失败
	// 各类类型被判定为「空（不满足 required）」的值
	// 类型			会触发 required 失败的值
	// string		"" 空字符串
	// int/int64	0
	// bool			false
	// slice/map	空切片、nil map
	// 指针			nil
}

type PullCategoryResp struct {
	CategoryName string `json:"CategoryName"`
	CategoryType int8   `json:"CategoryType"`
	Sort         int    `json:"sort"`
}

type PullFriendsReq struct {
	// url 查询参数 或 post 表单查询参数
	// GET ：/xxx?page=1&pageSize=10 url 后面问号的查询字符串
	// POST ：form‑urlencoded：Content‑Type:application/x‑www‑form‑urlencoded 表单提交
	Page     int `json:"page" form:"page" binding:"required"`
	PageSize int `json:"pageSize" form:"pageSize" binding:"required"`
}

type PullFriendsResp struct {
	Uid       string `json:"uid"`
	Nickname  string `json:"nickname"`
	AvatarUrl string `json:"avatarUrl"`
	IsOnline  bool   `json:"isOnline"`
}

type ChatResp struct {
	Uid             string `json:"uid"`
	SenderUID       string `json:"senderUid"`
	ReceiverUID     string `json:"receiverUid"`
	Nickname        string `json:"nickname"`
	AvatarUrl       string `json:"avatarUrl"`
	ConversationUID string `json:"conversationUid"`
	Content         string `json:"content"`
	IsInsertToTop   bool   `json:"isInsertToTop"`
}

type ChatReq struct {
	ConversationUID string `json:"conversationUid,omitempty"`
	ReceiverUID     string `json:"receiverUid" binding:"required"`
	Content         string `json:"content"`
	RequestId       string `json:"requestId,omitempty"` // 前端生成的请求ID，用于幂等性检查
}

type PullMessagesReq struct {
	PageSize        int    `json:"pageSize" form:"pageSize" binding:"required"`
	CursorMsgId     string `json:"cursorMsgId" form:"cursorMsgId"`
	ConversationUID string `json:"conversationUid" form:"conversationUid" binding:"required"`
}

type MarkConversationRead struct {
	ConversationUID string `json:"conversationUid" binding:"required"`
}
