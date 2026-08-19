package service

import (
	"ChatApp/dto"
	"ChatApp/socket"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
)

func ReplayAck(uid string, conn *websocket.Conn, requestId string, msgId string, success bool, errMsg string) error {
	if requestId == "" {
		return fmt.Errorf("requestId 为空，无需回复 ack")
	}
	payload := dto.AckRespDto{
		Success: success,
		ErrMsg:  errMsg,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	wc := &dto.WebsocketChange{
		MsgType:   "ack",
		MsgId:     msgId,
		RequestId: requestId,
		Data:      data,
	}
	return socket.WsInstance.SendSingleConn(uid, conn, wc)
}
