package lru

import (
	"container/list"
)

// cache LRU缓存
// 通过双向链表 + 哈希表实现 O(1) 访问与淘汰
type Cache struct {
	maxBytes int64
	nbytes   int64
	// 双向链表
	ll *list.List
	// 哈希表，key -> 链表节点（便于 O(1) 命中）
	cache map[string]*list.Element

	// 淘汰回调，可选
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

// Value 使用 Len() 方法返回占用的内存大小
type Value interface {
	Len() int
}

// New 创建一个新的 Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找功能 Get
// 如果 key 存在，则将对应节点移到队尾（表示最近使用过），并返回值
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		//双向链表作为队列，队首队尾是相对的，这里约定front为队尾
		c.ll.MoveToFront(ele)
		ret := ele.Value.(*entry)
		return ret.value, true
	}
	return
}

// 移除最近最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)

		// 从哈希表中删除
		delete(c.cache, kv.key)
		// 更新当前内存大小
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 调用回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 添加/更新功能 Add
// 若 key 已存在，则更新对应节点的值，并将该节点移到队尾
// 若 key 不存在，则新建节点并插入到队尾
// 添加新节点后，若超出最大内存限制，则移除最近最少访问的节点
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 若 key 不存在，新增节点
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())

	}
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// Len 返回当前缓存使用的内存大小
func (c *Cache) Len() int {
	return c.ll.Len()
}
