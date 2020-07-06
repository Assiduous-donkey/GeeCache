package geeCache

import (
	"geeCache/lru"
	"sync"
)

// 为单机缓存提供安全的并发操作
type cache struct {
	mux        sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.lru == nil {
		return
	}
	if value, ok := c.lru.Get(key); ok {
		return value.(ByteView), ok
	}
	return
}
func (c *cache) add(key string, value ByteView) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.lru == nil { // 延迟初始化
		c.lru = lru.NewCache(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}
