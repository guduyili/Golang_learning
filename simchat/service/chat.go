package service

import (
	"log"
	"simchat/model"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type ChatService struct {
	mu          sync.Mutex
	onlineConns map[string]*websocket.Conn // 记录在线用户连接状态
	historyMsgs []model.Message            // 存储历史消息
	userCounter int                        // 用户ID计数器
}

// NewChatService 创建一个新的 ChatService 实例
func NewChatService() *ChatService {
	return &ChatService{
		onlineConns: make(map[string]*websocket.Conn),
		historyMsgs: make([]model.Message, 0),
		userCounter: 0,
	}
}

// GenerateUserID 生成唯一的用户ID
func (s *ChatService) GenerateUserID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userCounter++
	return "user_" + string(rune(s.userCounter+'0'))
}

// AddOnlineUser 添加在线连接
func (s *ChatService) AddOnlineUser(userID string, conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onlineConns[userID] = conn
	log.Printf("用户上线： UserID=%s, 当前在线用户数=%d", userID, len(s.onlineConns))
}

// RemoveOnlineUser 移除离线连接
func (s *ChatService) RemoveOnlineUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.onlineConns, userID)
	log.Printf("用户下线： UserID=%s, 当前在线用户数=%d", userID, len(s.onlineConns))
}

// GetHistoryMessages 获取历史消息
func (s *ChatService) GetHistoryMessages() []model.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	history := make([]model.Message, len(s.historyMsgs))
	copy(history, s.historyMsgs)
	return history
}

// SaveMessage 保存聊天消息到历史记录
func (s *ChatService) SaveMessage(msg model.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	//限制历史消息数量，最多保存100条
	if len(s.historyMsgs) >= 100 {
		s.historyMsgs = s.historyMsgs[1:]
	}

	s.historyMsgs = append(s.historyMsgs, msg)
	log.Printf("保存消息：%s(%s) -> %s", msg.Name, msg.UserId, msg.Text)
}

// Broadcast 广播消息给所有在线用户
func (s *ChatService) Broadcast(senderUserId string, resp model.WsResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for userId, conn := range s.onlineConns {
		if userId == senderUserId {
			continue // 不发送给自己
		}

		// 发送JSON格式相应（超时控制 3秒）
		conn.SetWriteDeadline(time.Now().Add(3 * time.Second))

		if err := websocket.JSON.Send(conn, resp); err != nil {
			log.Printf("广播失败：UserId=%s，错误：%v", userId, err)
			// 发送失败视为连接失效，主动移除
			delete(s.onlineConns, userId)
		}
	}
}
