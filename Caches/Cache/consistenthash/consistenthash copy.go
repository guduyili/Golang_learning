package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

// Map 维护一致性哈希环：
// - replicas 表示虚拟节点数量，用于平衡数据分布
// - keys 保存所有虚拟节点的哈希值（升序）
// - hashMap 将虚拟节点哈希值映射回真实节点名称
type Map struct {
	hash     Hash
	replicas int
	keys     []int
	hashMap  map[int]string
}

// New 根据指定的虚拟节点数量和哈希函数创建 Map，若未指定哈希函数则默认使用 crc32.
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}

	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 将真实节点加入哈希环：
// - 为每个真实节点生成 replicas 个虚拟节点，命名为 i+key（i 是下标）
// - 对虚拟节点命名求哈希，插入有序切片 keys，并记录映射关系
// - 最后对 keys 排序，确保后续二分查找可用
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			//通过添加编号的方式区分不同虚拟节点。
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get 查找给定 key 对应的真实节点：
// - 先计算 key 的哈希值
// - 利用二分查找找到第一个 >= hash 的虚拟节点索引
// - 若越界则环状取模回到 0 号位置
// - 返回该虚拟节点映射的真实节点名称
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))

	// 通过二分查找，定位到第一个 >= hash 的虚拟节点索引
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// idx == len(m.keys) 说明 hash 大于所有虚拟节点哈希值，取环状的第一个节点
	return m.hashMap[m.keys[idx%len(m.keys)]]
}

//
