package service

import (
	"ChatApp/config"
	"ChatApp/dto"
	"ChatApp/global"
	"ChatApp/utils"
	"bytes"
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// proactiveSession 记录一个需要主动消息的会话状态
type proactiveSession struct {
	conversationUid string
	userUid         string
	aiUid           string
	lastActiveAt    time.Time // 用户最后一次发消息的时间
	lastProactiveAt time.Time // 最后一次主动消息的时间
}

// aiProactiveScheduler AI 主动消息调度器
type aiProactiveScheduler struct {
	db        *gorm.DB
	sessions  map[string]*proactiveSession // key: conversationUid
	mu        sync.RWMutex
	stopCh    chan struct{}
	httpClient *http.Client
}

var (
	proactiveScheduler *aiProactiveScheduler
	proactiveOnce      sync.Once
)

// InitAIProactiveScheduler 初始化并启动 AI 主动消息调度器
func InitAIProactiveScheduler(db *gorm.DB) {
	if !config.Conf.AI.ProactiveEnabled {
		global.Log.Info("AI 主动消息未开启，跳过调度器启动")
		return
	}
	proactiveOnce.Do(func() {
		proactiveScheduler = &aiProactiveScheduler{
			db:        db,
			sessions:  make(map[string]*proactiveSession),
			stopCh:    make(chan struct{}),
			httpClient: &http.Client{Timeout: 120 * time.Second},
		}
		go proactiveScheduler.run()
		global.Log.Info("AI 主动消息调度器已启动")
	})
}

// StopAIProactiveScheduler 停止调度器
func StopAIProactiveScheduler() {
	if proactiveScheduler != nil {
		close(proactiveScheduler.stopCh)
	}
}

// TrackAISession 记录一个 AI 会话（用户发消息给 AI 时调用）
func TrackAISession(conversationUid, userUid, aiUid string) {
	if proactiveScheduler == nil {
		return
	}
	proactiveScheduler.mu.Lock()
	defer proactiveScheduler.mu.Unlock()

	if s, ok := proactiveScheduler.sessions[conversationUid]; ok {
		s.lastActiveAt = time.Now()
		// 用户主动发消息后，重置主动消息计时器，避免刚聊完就发主动消息
		s.lastProactiveAt = time.Now()
	} else {
		proactiveScheduler.sessions[conversationUid] = &proactiveSession{
			conversationUid: conversationUid,
			userUid:         userUid,
			aiUid:           aiUid,
			lastActiveAt:    time.Now(),
			lastProactiveAt: time.Now(), // 初始化时不马上发，等一个间隔
		}
	}
}

// run 调度器主循环
func (s *aiProactiveScheduler) run() {
	// 从配置读取检查周期
	checkSec := config.Conf.AI.ProactiveCheckSec
	if checkSec <= 0 {
		checkSec = 600 // 默认 10 分钟
	}
	ticker := time.NewTicker(time.Duration(checkSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkAndSend()
		case <-s.stopCh:
			return
		}
	}
}

// checkAndSend 检查所有会话，对到时间的会话发送主动消息
func (s *aiProactiveScheduler) checkAndSend() {
	s.mu.RLock()
	var toSend []*proactiveSession
	now := time.Now()

	for _, sess := range s.sessions {
		// 该角色未开启主动消息则跳过
		if !isAIProactiveEnabled(sess.aiUid) {
			continue
		}
		// 从配置读取间隔
		minMin := config.Conf.AI.ProactiveMinMinute
		maxMin := config.Conf.AI.ProactiveMaxMinute
		if minMin <= 0 {
			minMin = 120 // 默认 2 小时
		}
		if maxMin <= minMin {
			maxMin = minMin + 60
		}
		minInterval := time.Duration(minMin) * time.Minute
		maxInterval := time.Duration(maxMin) * time.Minute
		// 用 conversationUid 做随机种子，保证每个会话间隔固定但不同
		rng := rand.New(rand.NewSource(hashString(sess.conversationUid)))
		interval := minInterval + time.Duration(rng.Float64()*float64(maxInterval-minInterval))

		if now.Sub(sess.lastProactiveAt) >= interval {
			// 复制一份，避免锁竞争
			toSend = append(toSend, &proactiveSession{
				conversationUid: sess.conversationUid,
				userUid:         sess.userUid,
				aiUid:           sess.aiUid,
				lastActiveAt:    sess.lastActiveAt,
				lastProactiveAt: sess.lastProactiveAt,
			})
		}
	}
	s.mu.RUnlock()

	for _, sess := range toSend {
		go s.sendProactive(sess)
	}
}

// sendProactive 给单个会话发送主动消息
func (s *aiProactiveScheduler) sendProactive(sess *proactiveSession) {
	aiServiceURL := config.Conf.AI.ServiceURL
	if aiServiceURL == "" {
		return
	}

	// 记录当前的 lastProactiveAt，用于后续二次检查（用户发消息会重置它）
	s.mu.RLock()
	originalProactiveAt := sess.lastProactiveAt
	s.mu.RUnlock()

	// 1. 调用 FastAPI /proactive 生成主动消息
	reqBody := map[string]string{
		"user_id":         sess.userUid,
		"ai_id":           sess.aiUid,
		"conversation_id": sess.conversationUid,
		"trigger_reason":  "定时主动发起话题",
	}
	body, _ := json.Marshal(reqBody)

	resp, err := s.httpClient.Post(aiServiceURL+"/proactive", "application/json", bytes.NewBuffer(body))
	if err != nil {
		global.Log.Warn("AI 主动消息生成失败", zap.String("conv", sess.conversationUid), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		global.Log.Warn("AI 主动消息生成返回非200", zap.Int("status", resp.StatusCode), zap.String("conv", sess.conversationUid))
		return
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Message == "" {
		global.Log.Warn("AI 主动消息解析失败", zap.String("conv", sess.conversationUid), zap.Error(err))
		return
	}

	// 2. 二次检查：调用 FastAPI 期间用户可能发了消息重置了 lastProactiveAt
	// 如果被重置了，说明用户正在活跃，不发送主动消息
	s.mu.RLock()
	current, ok := s.sessions[sess.conversationUid]
	shouldSend := ok && current.lastProactiveAt.Equal(originalProactiveAt)
	s.mu.RUnlock()
	if !shouldSend {
		global.Log.Info("AI 主动消息已取消（用户在此期间发了消息）", zap.String("conv", sess.conversationUid))
		return
	}

	// 3. 构造 AI 消息，入库并推送
	msgId := utils.GenAutoSnowId()
	chatReq := &dto.ChatReq{
		ConversationUID: sess.conversationUid,
		ReceiverUID:     sess.userUid,
		Content:         result.Message,
	}

	if _, err := CreateMessage(context.Background(), s.db, nil, sess.aiUid, chatReq, msgId); err != nil {
		global.Log.Error("AI 主动消息入库失败", zap.String("msgId", msgId), zap.Error(err))
		return
	}

	// 4. 更新最后主动消息时间
	s.mu.Lock()
	if s, ok := s.sessions[sess.conversationUid]; ok {
		s.lastProactiveAt = time.Now()
	}
	s.mu.Unlock()

	global.Log.Info("AI 主动消息已发送",
		zap.String("msgId", msgId),
		zap.String("from", sess.aiUid),
		zap.String("to", sess.userUid),
		zap.Int("contentLen", len(result.Message)),
	)
}

// hashString 简单的字符串哈希，用于随机种子
func hashString(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	return h
}

// isAIProactiveEnabled 判断某个 AI 角色是否开启了主动消息
func isAIProactiveEnabled(aiUid string) bool {
	for _, c := range config.Conf.AI.Characters {
		if c.UID == aiUid {
			return c.Proactive
		}
	}
	// 配置中没有的角色默认不开启主动消息
	return false
}
