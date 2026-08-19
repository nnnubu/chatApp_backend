package dto

import "encoding/json"

type WebsocketChange struct {
	MsgType   string          `json:"msgType"`             // 业务类型区分
	MsgId     string          `json:"msgId,omitempty"`     // 持久化消息 id 并同时作为 后端发送 ack 给前端确认依赖
	RequestId string          `json:"requestId,omitempty"` // 前端发送 ack 给后端确认依赖
	Data      json.RawMessage `json:"data"`
}

// ping			心跳测试
// chat			普通聊天
// addFriend	好友请求
// refuse		拒绝好友请求
// ack 			消息收发确认

type AckRespDto struct {
	Success bool   `json:"success"`
	ErrMsg  string `json:"errMsg,omitempty"`
}

type MarkReadDto struct {
	ReadType string `json:"readType"`
}
