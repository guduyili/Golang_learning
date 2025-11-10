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
	defer func() {
		if curUserId != "" {
			cs.RemoveOnlineUser(curUserId)
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
			handleJoin(conn, req, &curUserId)
		case "findAllMessages":
			handleFindAllMessages(conn)
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
func handleJoin(conn *websocket.Conn, req model.WsRequest, curUserId *string) {
	userName := strings.TrimSpace(req.Name)
	if userName == "" {
		sendResponse(conn, "typeFail", map[string]string{"msg": "请先登录"})
		return
	}

	userId := cs.GenerateUserID()
	*curUserId = userId
	cs.AddOnlineUser(userId, conn)

	sendResponse(conn, "joinSuccess", map[string]string{"userId": userId})

}

// handleFindAllMessages 处理获取历史消息请求
func handleFindAllMessages(conn *websocket.Conn) {
	history := cs.GetHistoryMessages()
	sendResponse(conn, "historyMessages", history)
	log.Printf("拉取历史消息：条数=%d", len(history))

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
	name, nameOk := msgData["name"].(string)
	timestamp, timeOk := msgData["timestamp"].(string)

	if !textOk {
		sendResponse(conn, "msgFail", map[string]string{"msg": "消息缺少text字段（需为字符串）"})
		return
	}
	if !nameOk {
		sendResponse(conn, "msgFail", map[string]string{"msg": "消息缺少name字段（需为字符串）"})
		return
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

	cs.SaveMessage(msg)

	cs.Broadcast(senderUserId, model.WsResponse{
		Type: "newMessage",
		Data: msg,
	})

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
	}

	isTyping, isTypingOk := typingData["isTyping"].(bool)
	if !isTypingOk {
		sendResponse(conn, "typeFail", map[string]string{"msg": "状态格式错误（isTyping需为布尔值）"})
		return
	}

	name, nameOK := typingData["name"].(string)
	if !nameOK {
		sendResponse(conn, "typeFail", map[string]string{"msg": "状态格式错误（name需为字符串）"})
		return
	}

	status := model.TypingData{
		IsTyping: isTyping,
		Name:     name,
		UserId:   senderUserId,
	}
	cs.Broadcast(senderUserId, model.WsResponse{
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
