package geecache

import (
	"fmt"
	"log"
	"mikucache/singleflight"
	"sync"
)

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
	// 使用singleflight.Group确保并发场景下针对相同的key，load过程只会调用一次
	loader *singleflight.Group
}

// 缓存不存在的时候，调用这个接口，获取源数据
type Getter interface {
	Get(key string) ([]byte, error)
}
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    singleflight.NewGroup(),
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	// 从mainCache中查找缓存，如果存在则返回缓存值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// mainCache中找不到就去load
	return g.load(key)
}
func (g *Group) load(key string) (value ByteView, err error) {
	// 每个key只请求一次 不管是本地还是远程
	// 并发场景下针对相同的key，load过程只会调用一次
	view, err := g.loader.Do(key, func() (any, error) {
		if g.peers != nil {
			// 先根据key选择对应的peer
			if peer, ok := g.peers.PickPeer(key); ok {
				// 然后从这个peer取出结果
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[MikuCache] Failed to get from peer", err)
			}
		}
		// 取本地的了
		return g.getLocally(key)
	})
	if err == nil {
		return view.(ByteView), nil
	}
	return
}


func (g *Group) getLocally(key string) (ByteView, error) {
	// 通过getter方法去获取key对应的value
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	// 将key和value添加到缓存中
	g.populateCache(key, value)
	return value, nil
}

// 将key和value添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {

	g.mainCache.add(key, value)
}

// 注册Peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	// 不能注册一次以上
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 从peer取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
