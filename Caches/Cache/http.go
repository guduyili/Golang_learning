package cache

import (
	consistenthash "cache/consistentHash"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// http://example.com/_cache/
const (
	defaultBasePath = "/_cache/"
	defaultReplicas = 50
)

type httpGetter struct {
	baseURL string
}

type HTTPPool struct {
	self string
	//作为节点间通讯地址的前缀
	basePath string

	//保护peers and httpGetters
	mu    sync.Mutex
	peers *consistenthash.Map
	//映射远程节点与对应的 httpGetter。
	// 每一个远程节点对应一个 httpGetter，
	// 因为 httpGetter 与远程节点的地址 baseURL 有关。
	httpGetters map[string]*httpGetter
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

// ServeHTTP 处理其他节点的拉取请求，路径格式为 /basePath/group/key
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

// Set 根据给定地址列表初始化一致性哈希环， 并未每个地址创建 httpGetter客户端
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{
			baseURL: peer + p.basePath,
		}
	}

}

// 包装了一致性哈希算法的 Get() 方法，
// 根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}

	return nil, false
}

var _PeerPicker = (*HTTPPool)(nil)

// Get 向目标节点发起HTTP请求以获取缓存数据
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 拼接请求地址： <peer-base>/<group>/<key>
	// 使用 url.QueryEscape 进行转义，避免特殊字符问题
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// 发起 GET 请求
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	// 关闭响应体
	defer res.Body.Close()

	//处理相应
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	// Read all
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// 编译期断言，确保 httpGetter 实现 PeerGetter 接口
var _PeerGetter = (*httpGetter)(nil)
