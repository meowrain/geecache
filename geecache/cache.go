package geecache

import (
	"mikucache/geecache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex // 保证并发安全
	lru        *lru.Cache // 缓存的核心数据结构
	cacheBytes int64      // 缓存的内存
}

func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil) //Lazy Initialization
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (ByteView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return ByteView{}, false
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return ByteView{}, false
}
