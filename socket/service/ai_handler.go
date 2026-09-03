package service

import (
	"ChatApp/config"
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/utils"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AI_UID 默认系统 AI 助手的 UID（需在数据库 user 表中存在对应记录）
const AI_UID = "ai_assistant_001"

// AI_UID_PREFIX 所有 AI 用户的 UID 前缀
const AI_UID_PREFIX = "ai_"

// IsAIUser 判断是否是 AI 用户（UID 以 ai_ 开头）
func IsAIUser(uid string) bool {
	return strings.HasPrefix(uid, AI_UID_PREFIX)
}

// aiChatRequest 发送给 FastAPI AI 服务的请求体
type aiChatRequest struct {
	UserID         string `json:"user_id"`
	AiID           string `json:"ai_id"`
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
}

// aiChatResponse FastAPI AI 服务的响应体
type aiChatResponse struct {
	Reply          string `json:"reply"`
	ConversationID string `json:"conversation_id"`
}

// HandleAIMessage 异步处理 AI 回复：
// 1. 调用 FastAPI 服务获取 AI 回复
// 2. 构造 AI 发送的消息（sender=AI, receiver=用户）
// 3. 复用 CreateMessage 入库并推送给用户
//
// 本函数应在 goroutine 中调用，不阻塞主消息流程。
// Go 端只做转发，并发控制和错误处理由 FastAPI 侧负责。
func HandleAIMessage(db *gorm.DB, senderUid string, chatReq *dto.ChatReq) {
	// 记录会话，用于主动消息调度
	TrackAISession(chatReq.ConversationUID, senderUid, chatReq.ReceiverUID)

	global.Log.Info("AI 消息处理开始",
		zap.String("from", senderUid),
		zap.String("to", chatReq.ReceiverUID),
		zap.String("content", chatReq.Content),
	)

	aiServiceURL := config.Conf.AI.ServiceURL
	if aiServiceURL == "" {
		global.Log.Warn("AI 服务地址未配置，跳过 AI 回复")
		return
	}

	// 1. 组装请求
	aiReq := aiChatRequest{
		UserID:         senderUid,
		AiID:           chatReq.ReceiverUID,
		ConversationID: chatReq.ConversationUID,
		Message:        chatReq.Content,
	}
	body, err := json.Marshal(aiReq)
	if err != nil {
		global.Log.Error("AI 请求序列化失败", zap.Error(err))
		return
	}

	// 2. 调用 FastAPI（超时设长一点，具体错误由 Python 侧处理）
	chatURL := aiServiceURL + "/chat"
	httpClient := &http.Client{Timeout: 180 * time.Second}
	resp, err := httpClient.Post(chatURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		global.Log.Error("调用 AI 服务失败", zap.String("url", chatURL), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		global.Log.Error("AI 服务返回非 200", zap.Int("status", resp.StatusCode))
		return
	}

	var aiResp aiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		global.Log.Error("解析 AI 响应失败", zap.Error(err))
		return
	}

	if aiResp.Reply == "" {
		global.Log.Warn("AI 回复为空")
		return
	}

	// 3. 构造 AI 回复消息，复用 CreateMessage 入库并推送
	aiMsgId := utils.GenAutoSnowId()
	aiChatReq := &dto.ChatReq{
		ConversationUID: chatReq.ConversationUID,
		ReceiverUID:     senderUid, // AI 发给用户
		Content:         aiResp.Reply,
		// AI 消息不设 RequestId，不参与幂等性检查
	}

	// 用独立 context，不依赖原请求的 ctx（原 ctx 可能已取消）
	// sender 用用户实际发消息给的那个 AI（chatReq.ReceiverUID），不是硬编码的 AI_UID
	if _, err := CreateMessage(context.Background(), db, nil, chatReq.ReceiverUID, aiChatReq, aiMsgId); err != nil {
		global.Log.Error("AI 消息入库失败", zap.String("msgId", aiMsgId), zap.Error(err))
		return
	}

	global.Log.Info("AI 消息已发送",
		zap.String("msgId", aiMsgId),
		zap.String("from", chatReq.ReceiverUID),
		zap.String("to", senderUid),
		zap.Int("contentLen", len(aiResp.Reply)),
	)
}
