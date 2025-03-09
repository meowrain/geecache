package consistenthash

import (
	"hash/crc32"
	"log"
	"sort"
	"strconv"
)

// Hash 定义哈希函数类型
type Hash func(data []byte) uint32

// Map 结构体用于实现一致性哈希
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个节点对应的虚拟节点数量
	keys     []int          // 存储所有虚拟节点的哈希值，并保持排序
	hashMap  map[int]string // 存储虚拟节点与真实节点的映射关系
}

// New 创建一个新的一致性哈希对象
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	// 如果未提供哈希函数，则默认使用 crc32.ChecksumIEEE
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 向一致性哈希环中添加节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 计算虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			// 记录虚拟节点和真实节点的映射
			m.hashMap[hash] = key
		}
	}
	// 对所有虚拟节点的哈希值进行排序，以便后续二分查找
	sort.Ints(m.keys)
}

// Get 根据键查找最近的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		log.Println("Warning: Trying to get from an empty hash ring")
		return ""
	}
	// 计算键的哈希值
	hash := int(m.hash([]byte(key)))
	// 使用二分查找找到第一个大于或等于 hash 的虚拟节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	/*
		若 idx == len(m.keys)，说明 hash 大于所有虚拟节点的哈希值，
		则选择第一个节点（形成环结构）。
	*/
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
