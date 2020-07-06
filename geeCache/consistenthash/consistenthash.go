package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func([]byte) uint32

type Map struct {
	keys     []int
	replicas int            // 真实节点的虚拟节点数量
	hashMap  map[int]string // 虚拟节点与真实节点的映射表
	hash     Hash
}

func NewMap(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE // 默认的哈希计算函数
	}
	return m
}

// 添加真实节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// 获取key所在的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	// 二分查找顺时针方向第一个>= key的hash值的节点
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]] // 映射到真实节点
}
