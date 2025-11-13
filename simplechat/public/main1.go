package main

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	socketio "github.com/googollee/go-socket.io"
	engineio "github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

// Message 描述一条聊天室消息。
// 包含 socket 连接 ID (ID)、昵称 (Name)、文本内容 (Text)、时间戳 (Timestamp)、消息类型 (Type)。
type Message struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
}

// CreateMessageDTO 对应前端发送过来的消息载荷。
// 服务端会补齐 ID 和 Name，因此这里只保留前端真正提供的字段。
type CreateMessageDTO struct {
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
}

// TypingPayload 用于广播“正在输入”状态，
// 包含触发该事件的用户昵称、socket ID 及是否正在输入。
type TypingPayload struct {
	Name     string `json:"name"`
	IsTyping bool   `json:"isTyping"`
	ID       string `json:"id"`
}

// ChatService 封装了消息存储和用户映射逻辑。
// - messages 保存历史消息（演示用内存 slice，可替换为数据库）。
// - clientToUser 记录 socket ID 与用户名的映射。
// - mu 保证并发访问安全（多个协程同时写入时避免数据竞争）。
type ChatService struct {
	mu           sync.Mutex
	messages     []Message
	clientToUser map[string]string
}

// NewChatService 创建一个初始化完毕的服务实例。
func NewChatService() *ChatService {
	return &ChatService{
		messages:     make([]Message, 0),
		clientToUser: make(map[string]string),
	}
}

// Create 处理新消息：补齐昵称、ID，生成 Message 并存入历史列表。
func (s *ChatService) Create(dto CreateMessageDTO, clientID string) Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 根据 socket ID 找到用户名
	name := s.clientToUser[clientID]

	// 若前端没传时间戳，服务端兜底生成一个 HH:mm 格式
	if dto.Timestamp == "" {
		dto.Timestamp = time.Now().Format("15:04")
	}

	msg := Message{
		ID:        clientID,
		Name:      name,
		Text:      dto.Text,
		Type:      dto.Type,
		Timestamp: dto.Timestamp,
	}

	// 追加到历史记录
	s.messages = append(s.messages, msg)
	return msg
}

// FindAll 返回一份历史消息的拷贝，避免调用方修改内部 slice。
func (s *ChatService) FindAll() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Identify 记录 socket ID 对应的用户名，并返回该 ID 供前端存储。
func (s *ChatService) Identify(name, clientID string) string {
	s.mu.Lock()
	s.clientToUser[clientID] = name
	s.mu.Unlock()
	return clientID
}

// Remove 在连接断开时注销该用户，避免残留映射导致后续广播报错。
func (s *ChatService) Remove(clientID string) {
	s.mu.Lock()
	delete(s.clientToUser, clientID)
	s.mu.Unlock()
}

// Typing 根据 socket ID 查用户名并构造一个 TypingPayload。
// 若用户尚未登记昵称，则返回 nil 代表忽略本次广播。
func (s *ChatService) Typing(isTyping bool, clientID string) *TypingPayload {
	s.mu.Lock()
	name := s.clientToUser[clientID]
	s.mu.Unlock()

	if name == "" {
		return nil
	}

	return &TypingPayload{
		Name:     name,
		IsTyping: isTyping,
		ID:       clientID,
	}
}

func main() {
	// 业务服务，用于处理消息逻辑与状态。
	service := NewChatService()

	// 创建 Socket.IO 服务器，指定可用的传输方式。
	// 这里显式允许 WebSocket 与轮询两种传输，并禁用跨域检查，方便开发调试。
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&websocket.Transport{
				CheckOrigin: func(*http.Request) bool { return true },
			},
			&polling.Transport{},
		},
	})

	// OnConnect 在客户端建立连接时触发。
	// 将客户端加入默认房间 main，便于统一广播。
	server.OnConnect("/", func(conn socketio.Conn) error {
		conn.Join("main")
		log.Printf("connected: %s", conn.ID())
		return nil
	})

	// createMessage 事件：前端发送新消息时触发。
	// 1. 使用服务逻辑补齐并保存消息
	// 2. 通过 Emit 和 BroadcastToRoom 向自己以及房间所有人推送
	server.OnEvent("/", "createMessage", func(conn socketio.Conn, dto CreateMessageDTO) Message {
		message := service.Create(dto, conn.ID())

		// 回发给自己（确认）
		conn.Emit("message", message)
		// 广播给房间其他成员
		server.BroadcastToRoom("/", "main", "message", message)
		return message
	})

	// findAllMessages 事件：请求历史消息列表。
	server.OnEvent("/", "findAllMessages", func(socketio.Conn) []Message {
		return service.FindAll()
	})

	// join 事件：用户输入昵称后调用，记录映射并把 socket ID 返回给前端。
	server.OnEvent("/", "join", func(conn socketio.Conn, payload struct {
		Name string `json:"name"`
	}, ack func(string)) {
		if payload.Name == "" {
			ack("")
			return
		}
		ack(service.Identify(payload.Name, conn.ID()))
	})

	// typing 事件：用户输入时会持续触发，广播给房间里除自己外的其他人。
	server.OnEvent("/", "typing", func(conn socketio.Conn, payload struct {
		IsTyping bool `json:"isTyping"`
	}) {
		if data := service.Typing(payload.IsTyping, conn.ID()); data != nil {
			server.BroadcastToRoom("/", "main", "typing", data)
		}
	})

	// Socket 级别的错误与断开事件，用于记录日志。
	server.OnError("/", func(conn socketio.Conn, err error) {
		if err == nil {
			return
		}
		if strings.Contains(err.Error(), "write: timeout") {
			// 与已断开的长轮询连接写超时，属正常情况，直接忽略
			return
		}
		log.Printf("socket error (%s): %v", safeConnID(conn), err)
	})
	server.OnDisconnect("/", func(conn socketio.Conn, reason string) {
		id := safeConnID(conn)
		if conn != nil {
			conn.LeaveAll()
		}
		if id != "<nil>" && id != "<unknown>" {
			service.Remove(id)
		}
		log.Printf("disconnected: %s (%s)", id, reason)
	})

	// 启动 Socket.IO 服务（需单独 goroutine）
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %v", err)
		}
	}()
	defer server.Close()

	// 将 /socket.io/ 路径挂到默认的 HTTP handler 上，
	// 并包裹一个简易的 CORS 中间件，允许所有来源访问。
	http.Handle("/socket.io/", cors(server))

	// 根路径仅返回一个简单字符串，可用来观察服务是否启动。
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("chat server running"))
	})

	log.Println("Listening on :3001")
	log.Fatal(http.ListenAndServe(":3001", nil))
}

// safeConnID 在连接已关闭或内部指针为空时，安全地返回连接 ID。
// 如果底层已经被释放，直接调用 conn.ID() 会触发 panic，因此这里做 recover。
func safeConnID(conn socketio.Conn) (id string) {
	if conn == nil {
		return "<nil>"
	}
	defer func() {
		if r := recover(); r != nil {
			id = "<unknown>"
		}
	}()
	return conn.ID()
}

// cors 是一个最简单的跨域中间件，允许任何来源访问指定 handler。
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 允许任意来源与基本请求头/方法
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		// 对预检 (OPTIONS) 请求直接返回 204
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// 继续执行后续处理
		next.ServeHTTP(w, r)
	})
}
