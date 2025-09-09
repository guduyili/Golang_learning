package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	// 在线用户列表
	OnlineMap map[string]*User
	// 保护在线用户的读写锁
	mapLock sync.RWMutex

	// 消息广播的channel
	Message chan string
}

// 创建一个server接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
	return server
}

// 监听Message广播消息的channel的goroutine，一旦有消息就发送给全部的在线User
func (s *Server) ListenMessager() {
	for {
		msg := <-s.Message
		// 将msg发送给全部的在线User
		s.mapLock.Lock()
		for _, cli := range s.OnlineMap {
			cli.C <- msg
		}
		s.mapLock.Unlock()

	}
}

// 广播消息的方法
func (s *Server) Broadcast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.Message <- sendMsg
}

// 处理用户连接业务
func (s *Server) Handler(conn net.Conn) {
	//当前连接的业务
	// fmt.Println("连接建立成功")

	user := NewUser(conn, s)

	// 新用户上线，将用户加入onlineMap中
	// s.mapLock.Lock()
	// s.OnlineMap[user.name] = user
	// s.mapLock.Unlock()
	user.Online()

	// 监听用户是否活跃的channel
	isLive := make(chan bool)

	// 广播当前用户上线消息
	// s.Broadcast(user, "已上线")

	// 接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				// s.Broadcast(user, "已下线")
				user.Offline()
				return
			}
			if err != nil {
				fmt.Println("Conn Read err:", err)
				return
			}

			// 提取用户的消息(去除'\n')
			msg := string(buf[:n-1])

			// 将得到的消息进行广播
			user.DoMessage(msg)

			// 用户的任意消息，代表当前用户是一个活跃的用户
			isLive <- true
		}
	}()

	//当前handler阻塞
	for {
		select {
		case <-isLive:
		// 当前用户是一个活跃用户，应该重置定时器
		// 不做任何事情，为了激活select，更新下面的定时器
		case <-time.After(time.Minute * 5):
			//已经超时
			//将当前用户强制下线
			user.SendMsg("你被踢了")
			// 销毁资源
			close(user.C)

			// 关闭连接
			conn.Close()
			// 退出当前Handler
			return
		}
	}

}

// 启动服务器的接口
func (s *Server) Start() {
	// sockert listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	// close listen socket
	defer listener.Close()

	// 启动监听Message的goroutine
	go s.ListenMessager()

	for {
		// 无限循环，持续接受连接
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			continue
		}
		go s.Handler(conn)
	}
}
