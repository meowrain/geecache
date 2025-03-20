package singleflight

import (
	"sync"
)

// 代表正在进行中，或者已经结束的请求，使用sync.WaitGroup锁避免重入
type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

// Group是singleflight的主数据结构，管理不同key的请求(call)
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func NewGroup() *Group {
	return &Group{}
}

/*
Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，
等待 fn 调用结束了，返回返回值或错误。
*/
// Do 方法确保对于相同的 key，函数 fn 只会被调用一次，并且所有对该 key 的调用都会等待这个唯一调用的结果。
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
