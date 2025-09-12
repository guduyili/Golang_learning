// main.go
package main

func main() {
	server := NewServer("127.0.0.1", 8090)

	// 启动TCP服务器（原有功能）
	go server.Start()

	// 启动WebSocket服务器（新功能）
	server.StartWebServer()
}
