package lru

import "container/list"

type Cache struct {
	maxBytes int64                    // 允许使用的最大内存
	nbytes   int64                    // 当前已经使用的内存
	deque    *list.List               // 双向链表
	cache    map[string]*list.Element // key-链表节点
	// 删除记录时执行的回调函数 可以为nil
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func NewCache(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nbytes:    0,
		deque:     list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查询功能
func (c *Cache) Get(key string) (value Value, ok bool) {
	if node, ok := c.cache[key]; ok {
		c.deque.MoveToFront(node)
		listEntry := node.Value.(*entry)
		return listEntry.value, true
	}
	return
}

// 缓存淘汰
func (c *Cache) Delete() {
	// 删除deque队尾(最近最少使用)的元素直到内存空间不溢出
	for c.nbytes >= c.maxBytes {
		node := c.deque.Back()
		if node == nil {
			break
		}
		c.deque.Remove(node)
		listEntry := node.Value.(*entry)
		delete(c.cache, listEntry.key)
		c.nbytes -= int64(len(listEntry.key)) + int64(listEntry.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(listEntry.key, listEntry.value)
		}
	}
}

// 新增/修改功能
func (c *Cache) Add(key string, value Value) {
	if node, ok := c.cache[key]; ok {
		// 更新键值对
		kv := node.Value.(*entry)
		kv.value = value
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		c.deque.MoveToFront(node)
	} else {
		// 新增键值对
		newNode := c.deque.PushFront(&entry{key, value})
		c.cache[key] = newNode
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	if c.nbytes >= c.maxBytes {
		c.Delete()
	}
}
func (c *Cache) Len() int {
	return c.deque.Len() // 键值对总量
}
