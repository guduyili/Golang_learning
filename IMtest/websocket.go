// websocket.go
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WebSocket用户结构
type WebSocketUser struct {
	*User
	wsConn *websocket.Conn
}

func NewWebSocketUser(conn *websocket.Conn, server *Server) *WebSocketUser {
	name := server.genUniqueName("User")
	base := &User{
		Name:   name,
		Addr:   conn.RemoteAddr().String(),
		C:      make(chan string, 32), // 加缓冲防止偶发阻塞
		server: server,
		// base.conn 置空，仅 TCP 用户使用
	}
	wu := &WebSocketUser{
		User:   base,
		wsConn: conn,
	}
	// 启动唯一写协程
	go wu.ListenMessage()
	return wu
}

// 覆盖发送：仅放入 channel，禁止直接写 wsConn
func (wu *WebSocketUser) SendMsg(msg string) {
	select {
	case wu.C <- msg:
	default:
		// 防止异常阻塞：如果满了丢弃或可改成阻塞
		log.Println("warn: websocket user channel full, drop msg")
	}
}

// 唯一写 wsConn 的协程
func (wu *WebSocketUser) ListenMessage() {
	for msg := range wu.C {
		if err := wu.wsConn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			log.Println("websocket write error:", err)
			return
		}
	}
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	user := NewWebSocketUser(conn, s)

	// 1. 先上线（加入OnlineMap并广播）
	user.Online()

	// 2. 立即发送分配的用户名
	user.SendMsg("您已分配用户名:" + user.Name)

	defer func() {
		user.Offline()
		close(user.C)
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			return
		}
		user.DoMessage(string(data))
	}
}

// 启动WebSocket服务
func (s *Server) StartWebServer() {
	http.HandleFunc("/ws", s.HandleWebSocket)
	exe, _ := os.Executable()
	base := filepath.Dir(exe)
	staticDir := filepath.Join(base, "static")
	http.Handle("/", http.FileServer(http.Dir(staticDir)))
	log.Println("WebSocket server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
