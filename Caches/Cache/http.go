package cache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// http://example.com/_cache/
const defaultBasePath = "/_cache/"

type HTTPPool struct {
	self string
	//作为节点间通讯地址的前缀
	basePath string
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打印服务器日志信息
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. 路径校验：必须以 basePath 开头，否则说明不是 geecache 的请求
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 2. 解析路径 期望格式 /<basePath>/<group>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)

	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 3. 根据 groupName 获取对应的 Group 实例
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 4. 读取缓存（内部会处理缓存命中/回源逻辑）
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. 返回二进制数据。ByteView.ByteSlice() 会生成一个新的拷贝，避免共享底层数组
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
