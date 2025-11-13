package main

import (
	"log"
	"net/http"
	"sync"

	socketio "github.com/googollee/go-socket.io"
	engineio "github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

// Message 描述一条聊天室信息
type Message struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
}

// ChatService 封装消息存储和用户映射逻辑
type ChatService struct {
	mu           sync.Mutex
	messages     []Message
	clientToUser map[string]string
}

// NewChatService 创建一个初始化完毕的服务实例
func NewChatService() *ChatService {
	return &ChatService{
		messages:     make([]Message, 0),
		clientToUser: make(map[string]string),
	}
}

// Identify 用户昵称和socket ID 映射
func (s *ChatService) Identify(name, clientID string) string {
	s.mu.Lock()
	s.clientToUser[clientID] = name
	s.mu.Unlock()
	log.Printf("用户映射绑定：clientID=%s -> name=%s", clientID, name) // 补充映射日志
	return clientID
}

// safeConnID 安全获取连接ID（避免连接已关闭时panic）
func safeConnID(conn socketio.Conn) (id string) {
	if conn == nil {
		return "<nil-conn>"
	}
	defer func() {
		if r := recover(); r != nil {
			id = "<panic-conn>"
			log.Printf("获取连接ID时发生panic：%v", r) // 补充panic日志
		}
	}()
	return conn.ID()
}

func main() {
	// 1. 初始化业务服务
	log.Println("初始化聊天业务服务...")
	service := NewChatService()

	// 2. 创建 Socket.IO 服务器（增强日志）
	log.Println("开始初始化 Socket.IO 服务器，注册传输方式：WebSocket + Polling")
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			&websocket.Transport{
				CheckOrigin: func(r *http.Request) bool {
					// 补充跨域请求日志，方便排查前端来源
					origin := r.Header.Get("Origin")
					log.Printf("收到 WebSocket 跨域请求，Origin: %s（已允许）", origin)
					return true // 开发环境允许所有跨域
				},
			},
			&polling.Transport{}, // 兼容不支持WebSocket的环境
		},
	})

	// 3. 捕获 Socket 全局错误（关键：定位连接过程中的错误）
	server.OnError("/", func(conn socketio.Conn, err error) {
		log.Printf("【Socket全局错误】clientID=%s，错误信息：%v", safeConnID(conn), err)
	})

	// 4. 处理客户端连接（增强日志：区分“尝试连接”和“连接成功”）
	server.OnConnect("/", func(conn socketio.Conn) error {
		clientID := safeConnID(conn)
		log.Printf("【客户端连接】clientID=%s 尝试建立连接...", clientID)

		// 简单连接有效性检查
		if conn == nil {
			errMsg := "连接对象为nil，连接失败"
			log.Printf("【客户端连接失败】clientID=%s，原因：%s", clientID, errMsg)
			return nil // 返回nil避免阻断其他逻辑，仅日志提示
		}

		log.Printf("【客户端连接成功】clientID=%s，连接已建立", clientID)
		return nil
	})

	// 5. 处理 join 事件（保留增强日志，确认事件接收和处理）
	server.OnEvent("/", "join", func(conn socketio.Conn, payload struct {
		Name string `json:"name"`
	}, ack func(string)) {
		clientID := safeConnID(conn)
		log.Printf("【收到join事件】clientID=%s，前端传参name：%s", clientID, payload.Name)

		// 昵称为空校验
		if payload.Name == "" {
			log.Printf("【处理join事件】clientID=%s，昵称为空，返回空ID", clientID)
			ack("")
			return
		}

		// 绑定昵称并返回clientID
		respID := service.Identify(payload.Name, clientID)
		log.Printf("【处理join事件成功】clientID=%s，绑定name=%s，返回respID=%s", clientID, payload.Name, respID)
		ack(respID)
	})

	// 6. 启动 Socket.IO 服务（单独协程，避免阻塞HTTP服务）
	go func() {
		log.Println("Socket.IO 服务开始监听请求...")
		if err := server.Serve(); err != nil {
			log.Fatalf("【Socket.IO服务启动失败】错误信息：%v（可能是传输方式配置错误或端口冲突）", err)
		}
	}()
	defer func() {
		log.Println("程序退出，关闭Socket.IO服务...")
		server.Close()
	}()

	// 7. 配置HTTP服务（承载Socket.IO连接，增强跨域日志）
	log.Println("初始化HTTP服务，配置跨域中间件...")
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 补充HTTP请求日志，确认请求来源和路径
			log.Printf("【收到HTTP请求】Method=%s，Path=%s，Origin=%s",
				r.Method, r.URL.Path, r.Header.Get("Origin"))

			// 跨域头配置
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")

			// 处理OPTIONS预检请求
			if r.Method == http.MethodOptions {
				log.Printf("【处理OPTIONS预检】Path=%s，返回204", r.URL.Path)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// 8. 挂载路由：Socket.IO路径 + 根路径提示
	http.Handle("/socket.io/", corsMiddleware(server)) // Socket.IO核心路由
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("【访问根路径】clientIP=%s，返回服务状态提示", r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("chat server running (join event ready) | 后端服务正常，可连接Socket.IO"))
	})

	// 9. 启动HTTP服务（监听3001端口）
	log.Println("HTTP服务准备启动，监听端口：3001...")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatalf("【HTTP服务启动失败】错误信息：%v（大概率是3001端口被占用，可更换端口重试）", err)
	}
}
