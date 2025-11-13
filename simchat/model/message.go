package model

// WsRequest 前端发送给后端的请求结构（统一格式）
type WsRequest struct {
	Type string      `json:"type"` // 事件类型（join/findAllMessages/createMessage/typing）
	Data interface{} `json:"data"` // 具体数据（根据 Type 动态解析）
	Name string      `json:"name"` // 仅 join 事件用（前端直接传 name）
}

// WsResponse 后端返回给前端的响应结构（统一格式）
type WsResponse struct {
	Type string      `json:"type"` // 响应类型（joinSuccess/historyMessages/newMessage等）
	Data interface{} `json:"data"` // 具体响应数据
}

// Message 聊天消息结构体（必须包含所有前端渲染所需字段）
type Message struct {
	Text      string `json:"text"`               // 消息内容（必填）
	Timestamp string `json:"timestamp"`          // 时间戳（HH:MM，必填）
	Name      string `json:"name"`               // 发送者昵称（必填）
	UserId    string `json:"userId"`             // 发送者ID（后端生成，必填）
	Room      string `json:"room,omitempty"`     // 所属房间（新增，多房间支持）
	ToUserId  string `json:"toUserId,omitempty"` // 私聊目标（新增，点对点）
}

// TypingData 正在输入状态结构体
type TypingData struct {
	IsTyping bool   `json:"isTyping"` // 是否正在输入（必填）
	Name     string `json:"name"`     // 用户名（必填）
	UserId   string `json:"userId"`   // 用户ID（必填）
}

// SystemEvent 系统通知结构体（用户加入/离开等）
type SystemEvent struct {
	UserId      string `json:"userId"`
	Name        string `json:"name"`
	Room        string `json:"room"`
	OnlineCount int    `json:"onlineCount"`
}
