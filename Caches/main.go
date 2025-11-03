package main

import (
	"cache"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"jw":    "114",
	"boyue": "514",
	"Sam":   "567",
}

func main() {
	// var URL = "http://localhost:9999/_cache/scores/Tom"
	// base := "/_cache/"
	// tmp := strings.SplitN(URL[len(base):], "/", 2)
	// fmt.Println(tmp)

	cache.NewGroup("test", 2, cache.GetterFunc(
		func(key string) ([]byte, error) {
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:6000"
	peers := cache.NewHTTPPool(addr)

	// 对外提供 HTTP 接口，路径格式 /_geecache/<group>/<key>
	log.Println("cache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))

}
