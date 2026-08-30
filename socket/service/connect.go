package service

import (
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/model"
	"ChatApp/socket"
	"ChatApp/utils"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func WebSocketConnect(c *gin.Context) {
	uidAny, hasUid := c.Get("uid")
	if !hasUid {
		c.JSON(200, model.CommonResp{
			Code:    401,
			Message: "登录态缺失，请重新登录",
		})
		return
	}

	uid, ok := uidAny.(string)
	if !ok {
		c.JSON(200, model.CommonResp{
			Code:    401,
			Message: "登录信息错误，请重新登录",
		})
		return
	}

	_, db, rc, err := utils.GetRequestSource(c)
	if err != nil {
		c.JSON(http.StatusOK, model.CommonResp{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	//c.Writer	用于服务器向客户端发送websocket握手响应（c.Write就是用来写响应数据内容的）
	//c.Request 用于接受客户端向服务器发送的请求
	//responseHeader 用于向客户端发送额外的响应头
	conn, err := socket.WsInstance.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	// 运行至此代表已被劫持成功，此后不要再尝试使用c.json写入 http 响应

	// 限制单条最大消息，防止攻击
	conn.SetReadLimit(4096)
	// 设置初始读取超时时间 防止静默断网永久阻塞读协程 静默断网例如 开飞行模式 关路由器电源
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	//使用channel存储消息,后面的缓冲大小一定要写，否则chan的发送和接受必须是同时的，会造成阻塞
	msgChan := make(chan []byte, 100)
	connCtx, cancel := context.WithCancel(context.Background())
	item := &socket.ConnectItem{
		MsgChan: msgChan,
		Ctx:     connCtx,
		Cancel:  cancel,
	}

	wsi := socket.WsInstance
	wsi.AddConn(uid, conn, item)

	// 阻塞连接断开 进行读操作 相当于一个生产者
	for {
		// messageType
		//常量名	数值	含义	使用场景
		//websocket.TextMessage		1	UTF8 文本帧	前后端业务消息、心跳 ping/pong
		//websocket.BinaryMessage	2	二进制帧	传输图片、文件、字节流等非文本数据
		//websocket.CloseMessage	8	关闭控制帧	前端 / 后端主动发送关闭连接指令，代表要断开 WS
		//websocket.PingMessage		9	Ping 心跳帧	协议底层心跳探测
		//websocket.PongMessage		10	Pong 心跳回复帧	收到 Ping 后回复的保活帧
		// 上面两个心跳是 gorilla/websocket 包自带的 TCP 链路保活 与业务的 心跳 是不同的
		msgType, message, err := conn.ReadMessage()
		// gorilla/websocket 特性：
		// 当 ReadMessage() 返回非 nil 错误后，底层 TCP 连接已经处于关闭状态
		// 不需要手动调用 conn.Close() 多次主动 Close 不会 panic，但属于多余操作

		// err != nil 网络异常、断连
		// msgType == websocket.CloseMessage：前端主动发送关闭报文，正常退出连接
		//if err != nil || msgType == websocket.CloseMessage {
		//	log.Printf("用户：%s 连接断开，清理资源中", uid)
		//	wsi.RemoveConn(uid, conn)
		//	return
		//}

		// msgType == websocket.CloseMessage 这个判断分支不再使用了 因为收到该类型的时候 ReadMessage() 一定会返回 error
		if err != nil {
			log.Printf("用户：%s 连接断开，清理资源中", uid)
			wsi.RemoveConn(uid, conn)
			return
		}

		// 每收到任意一条消息 重置超时计时器
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		// 忽略非文本消息
		if msgType != websocket.TextMessage {
			continue
		}

		var wc dto.WebsocketChange
		err = json.Unmarshal(message, &wc)
		if err != nil {
			log.Println("消息格式错误", string(message))
			continue
		}

		func() {
			// 为每条消息新建消息ctx 大循环内部单独闭包一个函数避免 msgCtx 无法回收
			msgCtx, msgCancel := socket.NewMsgContext(item.Ctx)
			defer msgCancel()

			switch wc.MsgType {
			case "ping":
				// 校验前端的心跳测试并返回 pong 防止前端重复连接消耗资源
				// 注意此处不可使用异步来进行 pong 的返回 否则可能造成前端 ping 堆积而造成 心跳判断异常
				errHeart := PushSingleMsg(uid, conn, "pong", "", nil)
				if errHeart != nil {
					log.Printf("回复pong失败: %v", errHeart)
				}
			case "chat":
				// 前端主动推送的消息不允许自行携带 msgId 只能由后端生成 此处直接生成而不采用 wc.MsgId

				/**
				聊天消息逻辑：
				消息卡片进入：首先用户在好友请求同意之后先吧请求消息加入消息表单，并生成 conversationUID 返回给前端
				前端拿到 数据渲染成聊天卡片 用户点击进去之后 自带聊天的基础字段数据 后续聊天发送 先入库和推送给目标用户

				好友主页进入： conversationUID 需由后端查询或生成并入库推送
				**/
				var errChat error
				var chatReq dto.ChatReq
				err = json.Unmarshal(wc.Data, &chatReq)
				if err != nil {
					ackErr := ReplayAck(uid, conn, wc.RequestId, wc.MsgId, false, "参数格式错误")
					if ackErr != nil {
						log.Printf("ack发送失败 reqId=%s err=%v", wc.RequestId, ackErr)
					}
					global.Log.Error(err.Error())
					break
				}
				// 传递前端生成的 requestId，用于幂等性检查
				chatReq.RequestId = wc.RequestId
				if chatReq.ReceiverUID == uid {
					ackErr := ReplayAck(uid, conn, wc.RequestId, wc.MsgId, false, "发送对象不能是自己")
					if ackErr != nil {
						log.Printf("ack发送失败 reqId=%s err=%v", wc.RequestId, ackErr)
					}
					break
				}
				newMsgId := utils.GenAutoSnowId()
				// 消息入库
				errChat = CreateMessage(msgCtx, db, rc, uid, &chatReq, newMsgId)
				if errChat != nil {
					log.Printf("有一条消息推送失败：: %v", errChat)
					ackErr := ReplayAck(uid, conn, wc.RequestId, wc.MsgId, false, errChat.Error())
					if ackErr != nil {
						log.Printf("ack发送失败 reqId=%s err=%v", wc.RequestId, ackErr)
					}
					break
				}
				// 再推送 ack
				ackErr := ReplayAck(uid, conn, wc.RequestId, newMsgId, true, "")
				if ackErr != nil {
					log.Printf("有一条消息接收失败：: %v", ackErr)
					break
				}
			case "markRead":
				var readData dto.MarkReadDto
				err = json.Unmarshal(wc.Data, &readData)
				if err != nil {
					ackErr := ReplayAck(uid, conn, wc.RequestId, wc.MsgId, false, "参数格式错误")
					if ackErr != nil {
						log.Printf("ack发送失败 reqId=%s err=%v", wc.RequestId, ackErr)
					}
					global.Log.Error(err.Error())
					break
				}
				var errMarkRead error
				switch readData.ReadType {
				case "friendApply":
					errMarkRead = MarkFriendApplyRead(msgCtx, db, uid, wc.MsgId)
				case "chat":
					errMarkRead = errors.New("测试中")
				default:
					errMarkRead = errors.New("不支持的readType")
				}

				if errMarkRead != nil {
					global.Log.Error(errMarkRead.Error())
					_ = ReplayAck(uid, conn, wc.RequestId, wc.MsgId, false, errMarkRead.Error())
					break
				}
				_ = ReplayAck(uid, conn, wc.RequestId, wc.MsgId, true, "")
			default:
				log.Println(string(message))
				log.Println("未知业务类型", wc.MsgType)
			}
		}()
	}
}
