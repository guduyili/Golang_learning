package service

import (
	"fmt"
	"log"
	"simchat/model"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type ChatService struct {
	mu          sync.Mutex
	onlineConns map[string]*websocket.Conn     // 在线用户连接
	historyMsgs map[string][]model.Message     // 房间 -> 历史消息列表
	userCounter int                            // 用户ID计数器
	userNames   map[string]string              // userId -> name 映射
	rooms       map[string]map[string]struct{} // room -> set(userId)
	userRooms   map[string]string              // userId -> room
	nameIndex   map[string]string              // name -> userId（防重复登录）
}

// NewChatService 创建一个新的 ChatService 实例
func NewChatService() *ChatService {
	return &ChatService{
		onlineConns: make(map[string]*websocket.Conn),
		historyMsgs: make(map[string][]model.Message),
		userNames:   make(map[string]string),
		rooms:       make(map[string]map[string]struct{}),
		userRooms:   make(map[string]string),
		nameIndex:   make(map[string]string),
	}
}

// GenerateUserID 生成唯一的用户ID
func (s *ChatService) GenerateUserID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userCounter++
	return "user_" + strconv.Itoa(s.userCounter)
}

// AddOnlineUser 添加在线连接
func (s *ChatService) AddOnlineUser(userID, name, room string, conn *websocket.Conn) (onlineCount int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if old, exists := s.nameIndex[name]; exists && old != "" {
		return 0, fmt.Errorf("name in use")
	}
	s.onlineConns[userID] = conn
	s.userNames[userID] = name
	s.userRooms[userID] = room
	s.nameIndex[name] = userID
	if _, ok := s.rooms[room]; !ok {
		s.rooms[room] = make(map[string]struct{})
	}
	s.rooms[room][userID] = struct{}{}
	onlineCount = len(s.rooms[room])
	log.Printf("用户上线： UserID=%s Name=%s Room=%s 房间在线=%d", userID, name, room, onlineCount)
	return onlineCount, nil

}

// RemoveOnlineUser 移除离线连接
func (s *ChatService) RemoveOnlineUser(userID string) (name string, roomsLeft []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.onlineConns, userID)

	name = s.userNames[userID]
	delete(s.userNames, userID)

	if name != "" {
		delete(s.nameIndex, name)
	}
	if room, ok := s.userRooms[userID]; ok {
		roomsLeft = append(roomsLeft, room)
		delete(s.userRooms, userID)
	}

	for _, room := range roomsLeft {
		if members, ok := s.rooms[room]; ok {
			delete(members, userID)
			if len(members) == 0 {
				delete(s.rooms, room)
			}
		}
	}

	log.Printf("用户下线： UserID=%s Name=%s 退出房间=%v", userID, name, roomsLeft)
	return
}

// GetHistoryMessages 获取历史消息
func (s *ChatService) GetHistoryMessages(room string) []model.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgs := s.historyMsgs[room]
	history := make([]model.Message, len(msgs))
	copy(history, msgs)
	return history
}

// SaveMessage 保存聊天消息到历史记录
func (s *ChatService) SaveMessage(msg model.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room := msg.Room
	if _, ok := s.historyMsgs[room]; !ok {
		s.historyMsgs[room] = make([]model.Message, 0)
	}

	lst := s.historyMsgs[room]
	if len(lst) >= 100 {
		lst = lst[1:]
	}
	lst = append(lst, msg)
	s.historyMsgs[room] = lst
	log.Printf("保存消息：房间=%s %s(%s) -> %s", room, msg.Name, msg.UserId, msg.Text)
}

// BroadcastRoom 在指定房间广播（不含发送者）
func (s *ChatService) BroadcastRoom(room, senderUserId string, resp model.WsResponse) {
	s.mu.Lock()

	members := s.rooms[room]
	snapshot := make([]struct {
		id   string
		conn *websocket.Conn
	}, 0, len(members))

	for uid := range members {
		if uid == senderUserId {
			continue
		}

		if conn, ok := s.onlineConns[uid]; ok {
			snapshot = append(snapshot, struct {
				id   string
				conn *websocket.Conn
			}{id: uid, conn: conn})
		}
	}
	s.mu.Unlock()

	for _, item := range snapshot {
		item.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if err := websocket.JSON.Send(item.conn, resp); err != nil {
			log.Printf("广播失败：Room=%s UserId=%s err=%v", room, item.id, err)
			s.mu.Lock()
			// 从在线连接表移除
			delete(s.onlineConns, item.id)
			if members != nil {
				// 从房间成员集合移除
				delete(members, item.id)
			}
			s.mu.Unlock()
		}
	}
}

// SendPrivate 发送私聊消息
func (s *ChatService) SendPrivate(toUserId string, resp model.WsResponse) {
	s.mu.Lock()
	conn := s.onlineConns[toUserId]
	s.mu.Unlock()
	if conn == nil {
		return
	}
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_ = websocket.JSON.Send(conn, resp)
}

// GetUserName 获取用户昵称
func (s *ChatService) GetUserName(userId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userNames[userId]
}

// GetUserRoom 获取用户所在房间
func (s *ChatService) GetUserRoom(userId string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userRooms[userId]
}
