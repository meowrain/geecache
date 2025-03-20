package singleflight

import "sync"

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

/*
Do 的作用就是，针对相同的 key，无论 Do 被调用多少次，函数 fn 都只会被调用一次，
等待 fn 调用结束了，返回返回值或错误。
*/
// Do 方法确保对于相同的 key，函数 fn 只会被调用一次，并且所有对该 key 的调用都会等待这个唯一调用的结果。
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// 加锁以确保对共享资源 g.m 的并发安全访问
	g.mu.Lock()
	// 检查 g.m 是否为空，如果为空则初始化它
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 检查是否已经有针对该 key 的请求正在进行
	if c, ok := g.m[key]; ok {
		// 如果有，释放锁以允许其他请求继续
		g.mu.Unlock()
		// 等待该请求完成
		c.wg.Wait()
		// 返回已完成请求的结果
		return c.val, c.err
	}
	// 如果没有针对该 key 的请求正在进行，创建一个新的 call 实例
	c := new(call)
	// 增加 WaitGroup 的计数，表示有一个新的请求正在进行
	c.wg.Add(1)
	// 将新的 call 实例添加到 g.m 中
	g.m[key] = c
	// 释放锁以允许其他请求继续
	g.mu.Unlock()

	// 调用传入的函数 fn，并将结果存储在 call 实例中
	c.val, c.err = fn()
	// 减少 WaitGroup 的计数，表示该请求已完成
	c.wg.Done()

	// 再次加锁以确保对共享资源 g.m 的并发安全访问
	g.mu.Lock()
	// 从 g.m 中删除该 key 的记录，因为请求已经完成
	delete(g.m, key)
	// 释放锁以允许其他请求继续
	g.mu.Unlock()

	// 返回函数 fn 的结果
	return c.val, c.err
}
