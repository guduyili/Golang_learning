package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	server *Server
}

// 创建一个用户的API
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	// 使用随机用户名，保持与WebSocket一致
	name := server.genUniqueName("TCPUser")

	user := &User{
		Name:   name, // 改：使用随机名而不是IP
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	go user.ListenMessage()
	return user
}

// 用户的上线业务
func (u *User) Online() {
	//用户上线，将用户加入onlineMap中
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()

	//广播用户上线消息
	u.server.Broadcast(u, "已上线")

}

// 用户的下线业务
func (u *User) Offline() {
	//用户下线，将用户从onlineMap中删除
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()

	//广播用户下线消息
	u.server.Broadcast(u, "已下线")
}

// 给当前User对应的客户端发送消息（统一入口）
func (u *User) SendMsg(msg string) {
	// 对 TCP 用户：消息放入管道，ListenMessage 负责写入真实连接
	u.C <- msg
}

// 用户处理消息业务
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		// 确保返回所有在线用户（包括自己）
		u.server.mapLock.RLock()
		for _, user := range u.server.OnlineMap {
			// 去掉末尾换行，让前端统一处理
			onlinemsg := "[" + user.Addr + "]" + user.Name + ":" + "在线..."
			u.SendMsg(onlinemsg)
		}
		u.server.mapLock.RUnlock()
		return
	}

	if len(msg) > 7 && strings.HasPrefix(msg, "rename|") {
		// rename|新名
		newName := strings.SplitN(msg, "|", 2)[1]
		newName = strings.TrimSpace(newName)
		if newName == "" {
			u.SendMsg("用户名不能为空")
			return
		}

		u.server.mapLock.Lock()
		if _, exists := u.server.OnlineMap[newName]; exists {
			u.server.mapLock.Unlock()
			u.SendMsg("当前用户名已被使用")
			return
		}
		delete(u.server.OnlineMap, u.Name)
		u.server.OnlineMap[newName] = u
		u.server.mapLock.Unlock()

		u.Name = newName
		u.SendMsg("您已更新用户名:" + u.Name)
		u.server.Broadcast(u, "改名为:"+u.Name)
		return
	}

	if len(msg) > 4 && strings.HasPrefix(msg, "to|") {
		// to|对方|内容（内容可含“|”）
		parts := strings.SplitN(msg, "|", 3)
		if len(parts) < 3 {
			u.SendMsg("消息格式不正确，请使用\"to|张三|消息内容\"")
			return
		}
		remoteName := strings.TrimSpace(parts[1])
		if remoteName == "" {
			u.SendMsg("消息格式不正确，请使用\"to|张三|消息内容\"")
			return
		}

		u.server.mapLock.RLock()
		remoteUser, ok := u.server.OnlineMap[remoteName]
		u.server.mapLock.RUnlock()
		if !ok {
			u.SendMsg("该用户名不存在")
			return
		}

		content := parts[2]
		if strings.TrimSpace(content) == "" {
			u.SendMsg("无消息内容，请重发")
			return
		}
		remoteUser.SendMsg(u.Name + "对您说:" + content)
		return
	}

	// 其它消息走广播
	u.server.Broadcast(u, msg)
}

// TCP 用户的监听（WebSocketUser 已覆盖，不会调用这里）
func (u *User) ListenMessage() {
	if u.conn == nil {
		return
	}
	for msg := range u.C {
		_, _ = u.conn.Write([]byte(msg + "\n"))
	}
}
