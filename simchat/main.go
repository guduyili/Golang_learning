package main

import (
	"log"
	"net/http"
	sc "sc"
	"simchat/model"
	"simchat/service"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var cs = service.NewChatService()

// wsHandler 保持原有签名（被 websocket.Handler 包装并在 sc 的路由中调用）
func wsHandler(conn *websocket.Conn) {
	var curUserId string
	var curRoom string
	defer func() {
		if curUserId != "" {
			name, rooms := cs.RemoveOnlineUser(curUserId)

			for _, room := range rooms {
				cs.BroadcastRoom(room, curUserId, model.WsResponse{
					Type: "userLeave",
					Data: model.SystemEvent{UserId: curUserId, Name: name, Room: room},
				})
			}
		}
		conn.Close()
		log.Printf("连接关闭：UserId=%s，远程地址：%s", curUserId, conn.RemoteAddr())
	}()

	log.Printf("新连接建立：远程地址：%s", conn.RemoteAddr())

	for {
		var req model.WsRequest
		if err := websocket.JSON.Receive(conn, &req); err != nil {
			log.Printf("接收消息失败：%v", err)
			break
		}

		switch req.Type {
		case "join":
			handleJoin(conn, req, &curUserId, &curRoom)
		case "findAllMessages":
			handleFindAllMessages(conn, curUserId)
		case "createMessage":
			handleCreateMessage(conn, req, curUserId)
		case "typing":
			handleTyping(conn, req, curUserId)
		default:
			log.Printf("未知请求类型：UserId=%s，Type=%s", curUserId, req.Type)
			sendResponse(conn, "unknownType", map[string]string{"msg": "未知请求类型"})
		}
	}

}

// handleJoin 处理用户加入聊天室请求
func handleJoin(conn *websocket.Conn, req model.WsRequest, curUserId *string, curRoom *string) {
	userName := strings.TrimSpace(req.Name)
	if userName == "" {
		sendResponse(conn, "joinFail", map[string]string{"msg": "昵称不能为空", "code": "emptyName"})
		return
	}

	// 已登录用户重复 join，直接拒绝，避免重复映射与广播
	if *curUserId != "" {
		sendResponse(conn, "joinFail", map[string]string{"msg": "已登录，如需切换请先断开"})
		return
	}

	// （删除错误的重复已登录判断）

	// 可选房间参数：若 Data 中包含 {"room":"..."} 则加入该房间，否则默认 lobby
	room := "lobby"
	if req.Data != nil {
		if m, ok := req.Data.(map[string]interface{}); ok {
			if r, ok := m["room"].(string); ok && strings.TrimSpace(r) != "" {
				room = strings.TrimSpace(r)
			}
		}
	}

	userId := cs.GenerateUserID()
	*curUserId = userId

	// 绑定在线用户（维护 name 与 room 映射），并返回当前房间在线人数
	onlineCount, err := cs.AddOnlineUser(userId, userName, room, conn)
	if err != nil {
		// 名称已被占用
		sendResponse(conn, "joinFail", map[string]string{"msg": "昵称已被占用", "code": "nameInUse"})
		*curUserId = ""
		return
	}
	*curRoom = room

	// 给自己返回 join 成功，附带房间在线人数
	sendResponse(conn, "joinSuccess", map[string]interface{}{"userId": userId, "name": userName, "room": room, "onlineCount": onlineCount})

	// 向同房间其他尘缘广播有人加入
	cs.BroadcastRoom(room, userId, model.WsResponse{Type: "userJoin", Data: model.SystemEvent{UserId: userId, Name: userName, Room: room, OnlineCount: onlineCount}})
}

// handleFindAllMessages 处理获取历史消息请求（需已登录）
func handleFindAllMessages(conn *websocket.Conn, curUserId string) {
	if curUserId == "" {
		sendResponse(conn, "historyFail", map[string]string{"msg": "请先登录"})
		return
	}

	room := cs.GetUserRoom(curUserId)
	history := cs.GetHistoryMessages(room)
	sendResponse(conn, "historyMessages", history)
	log.Printf("拉取历史消息：UserId=%s Room=%s 条数=%d", curUserId, room, len(history))
}

// handleCreateMessage 处理发送消息（createMessage）请求
func handleCreateMessage(conn *websocket.Conn, req model.WsRequest, senderUserId string) {
	if senderUserId == "" {
		sendResponse(conn, "msgFail", map[string]string{"msg": "请先登录"})
		return
	}

	msgData, ok := req.Data.(map[string]interface{})
	if !ok {
		sendResponse(conn, "msgFail", map[string]string{"msg": "消息格式错误（需为对象）"})
		return
	}

	text, textOk := msgData["text"].(string)
	// 服务器统一填充 name，避免信任前端随意冒用昵称
	name := cs.GetUserName(senderUserId)
	timestamp, timeOk := msgData["timestamp"].(string)
	toUserId, _ := msgData["toUserId"].(string)

	if !textOk {
		sendResponse(conn, "msgFail", map[string]string{"msg": "消息缺少text字段（需为字符串）"})
		return
	}
	if strings.TrimSpace(name) == "" {
		name = "匿名用户"
	}
	if !timeOk {
		sendResponse(conn, "msgFail", map[string]string{"msg": "消息缺少timestamp字段（需为字符串）"})
		return
	}

	if strings.TrimSpace(text) == "" {
		sendResponse(conn, "msgFail", map[string]string{"msg": "消息内容不能为空"})
		return
	}

	msg := model.Message{
		Text:      text,
		Name:      name,
		UserId:    senderUserId,
		Timestamp: timestamp,
	}

	if strings.TrimSpace(toUserId) != "" {
		//私聊，进发送给目标，对方不在线则静默（不存历史）
		msg.ToUserId = toUserId
		cs.SendPrivate(toUserId, model.WsResponse{
			Type: "privateMessage",
			Data: msg,
		})
	} else {
		//房间消息，根据用户当前room广播并保存历史
		msg.Room = cs.GetUserRoom(senderUserId)
		cs.SaveMessage(msg)
		cs.BroadcastRoom(msg.Room, senderUserId, model.WsResponse{Type: "newMessage", Data: msg})
	}

	sendResponse(conn, "msgSuccess", map[string]string{"msg": "消息发送成功"})
}

// handleTyping 处理用户输入状态变化
func handleTyping(conn *websocket.Conn, req model.WsRequest, senderUserId string) {
	if senderUserId == "" {
		sendResponse(conn, "typingFail", map[string]string{"msg": "请先登录"})
		return
	}

	typingData, ok := req.Data.(map[string]interface{})
	if !ok {
		sendResponse(conn, "typeFail", map[string]string{"msg": "状态格式错误（需为对象）"})
		return
	}

	isTyping, isTypingOk := typingData["isTyping"].(bool)
	if !isTypingOk {
		sendResponse(conn, "typeFail", map[string]string{"msg": "状态格式错误（isTyping需为布尔值）"})
		return
	}

	// 统一从服务器侧获取昵称
	name := cs.GetUserName(senderUserId)
	if strings.TrimSpace(name) == "" {
		name = "匿名"
	}

	status := model.TypingData{
		IsTyping: isTyping,
		Name:     name,
		UserId:   senderUserId,
	}
	// 仅向同房间成员广播输入状态
	room := cs.GetUserRoom(senderUserId)
	cs.BroadcastRoom(room, senderUserId, model.WsResponse{
		Type: "typingStatus",
		Data: status,
	})
	sendResponse(conn, "typingSuccess", map[string]string{"msg": "状态已上报"})

}

// sendResponse 通用相应发送参数
func sendResponse(conn *websocket.Conn, respType string, data interface{}) {
	resp := model.WsResponse{
		Type: respType,
		Data: data,
	}
	// 写入设置 3 秒超时，避免网络阻塞拖垮会话
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err := websocket.JSON.Send(conn, resp); err != nil {
		log.Printf("发送响应失败：Type=%s，Error=%v", respType, err)
	}
}

func main() {
	// 创建一个新的引擎实例
	r := sc.Default()

	// 根路由
	r.GET("/", func(c *sc.Context) {
		c.String(http.StatusOK, "Simplechat Goserver| 端口：3001")
	})

	// ws路由
	r.GET("/ws", func(c *sc.Context) {
		websocket.Handler(wsHandler).ServeHTTP(c.Writer, c.Req)
	})

	log.Println("Simplechat Goserver 服务启动，监听端口：3001，WebSocket 路径：/ws")

	if err := r.Run(":3001"); err != nil {
		log.Fatalf("服务启动失败：%v", err)
	}
}
