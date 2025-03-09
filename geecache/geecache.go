package geecache

import (
	"fmt"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

// 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

/*
一个 Group 可以认为是一个缓存的命名空间，每个 Group 拥有一个唯一的名称 name。比如可以创建三个 Group，缓存学生的成绩命名为 scores，缓存学生信息的命名为 info，缓存学生课程的命名为 courses。
*/
type Group struct {
	name      string //命名
	getter    Getter //getter Getter，即缓存未命中时获取源数据的回调(callback)。
	mainCache cache  // 缓存
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// mainCache中存在，直接获取返回
	if v, ok := g.mainCache.get(key); ok {
		return v, nil
	}
	// 如果不存在，加载
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	// 从本地获取数据
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{
		b: cloneBytes(bytes),
	}
	g.populateCache(key, value)
	return value, nil
}

// 将源数据添加到缓存 mainCache 中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
