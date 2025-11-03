// 并发控制
package cache

import (
	"cache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex // 保护并发访问的互斥锁
	lru        *lru.Cache // 底层的 LRU 缓存
	cacheBytes int64      // 最大缓存大小
}

// add
func (c *cache) add(key string, value lru.Value) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// get
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), true
	}

	return
}
