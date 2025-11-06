package main

import (
	"cache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"jw":    "114",
	"boyue": "514",
	"Sam":   "567",
}

func createGroup() *cache.Group {
	return cache.NewGroup("test", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// startCacheServer 启动缓存服务器
func startCacheServer(addr string, addrs []string, g *cache.Group) {
	peers := cache.NewHTTPPool(addr)
	peers.Set(addrs...)

	g.RegisterPeers(peers)

	log.Println("cache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))

}

// startAPIServer 启动一个 API 服务器，供用户访问
func startAPIServer(apiAddr string, g *cache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := g.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	// var URL = "http://localhost:9999/_cache/scores/Tom"
	// base := "/_cache/"
	// tmp := strings.SplitN(URL[len(base):], "/", 2)
	// fmt.Println(tmp)

	// cache.NewGroup("test", 2, cache.GetterFunc(
	// 	func(key string) ([]byte, error) {
	// 		if v, ok := db[key]; ok {
	// 			return []byte(v), nil
	// 		}
	// 		return nil, fmt.Errorf("%s not exist", key)
	// 	}))

	// addr := "localhost:6000"
	// peers := cache.NewHTTPPool(addr)

	// // 对外提供 HTTP 接口，路径格式 /_geecache/<group>/<key>
	// log.Println("cache is running at", addr)
	// log.Fatal(http.ListenAndServe(addr, peers))

	// 定义命令行参数：
	// -port：缓存服务器端口（默认8001）
	// -api：是否启动API服务器（默认false）
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Cache server port")
	flag.BoolVar(&api, "api", false, "start api server")
	flag.Parse()

	apiAddr := "http://localhost:4000"

	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	g := createGroup()
	if api {
		go startAPIServer(apiAddr, g)
	}
	startCacheServer(addrMap[port], addrs, g)
}
