// 负责与外部交互，控制缓存存储和获取的主流程

package cache

import (
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 创建一个新的缓存组
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
	}

	groups[name] = g
	return g
}

// GetGroup 根据名称获取已创建的Group，若不存在则返回nil
func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()

	g := groups[name]
	return g
}

// Get 方法实现了上述所说的流程 ⑴ 和 ⑶。
// 流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
// 流程 ⑶ ：缓存不存在，则调用 load 方法，load 调用 getLocally
// （分布式场景下会调用 getFromPeer 从其他节点获取），
// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，
// 并且将源数据添加到缓存 mainCache 中（通过 populateCache 方法
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, nil
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[Cache] hit")
		return v, nil
	}

	return g.load(key)
}

// load 表示“从源头加载数据”
func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

// getLocally 使用回调函数获取数据并添加到缓存
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
