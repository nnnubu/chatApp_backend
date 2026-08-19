package service

import (
	"ChatApp/dto"
	"ChatApp/socket"
	"encoding/json"
	"errors"

	"github.com/gorilla/websocket"
)

// PushBroadCastMsg 将消息广播到用户的所有在线设备
func PushBroadCastMsg(targetUid string, msgType string, msgId string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.New("解析数据出错")
	}
	wc := &dto.WebsocketChange{
		MsgType: msgType,
		MsgId:   msgId,
		Data:    data,
	}
	// 虽然 SendAllConnOfUid 又执行了一次但是如果直接把 结构体 传递给 Data
	// 就会递归序列化 结构体 产生大量转义字符 前端可能还需要使用 json.parse 去掉转义字符
	err = socket.WsInstance.SendAllConnOfUid(targetUid, wc)
	if err != nil {
		return err
	}
	return nil
}

// PushSingleMsg 将消息发送给指定连接
func PushSingleMsg(targetUid string, conn *websocket.Conn, msgType string, msgId string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return errors.New("解析数据出错")
	}
	wc := &dto.WebsocketChange{
		MsgType: msgType,
		MsgId:   msgId,
		Data:    data,
	}
	err = socket.WsInstance.SendSingleConn(targetUid, conn, wc)
	if err != nil {
		return err
	}
	return nil
}
