package geecache

import (
	"fmt"
	"log"
	"sync"
)

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
}

type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

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
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPiker called more than once")
	}
	g.peers = peers
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	//
	if g.peers != nil {
		// 调用实现了 PeerPicker 接口的 HTTPPool 注入的一致性哈希算法的 Get() 方法，
		// 根据具体的 key，选择节点，返回节点对应的 HTTP 客户端。
		// 通过 HTTP 客户端访问远程节点，获取缓存值。
		// 如果访问失败，执行本地的 getLocally() 方法获取。
		// 如果访问成功，将缓存值添加到本地缓存中。
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
		}
	}
	return g.getLocally(key)
}

// getFromPeer 从远程节点获取缓存值。
// 该方法接收一个实现了 PeerGetter 接口的 peer 对象和一个 key 作为参数。
// 它尝试从指定的远程节点获取与 key 对应的缓存值。
// 如果获取成功，返回一个 ByteView 类型的缓存值；如果出现错误，返回一个空的 ByteView 和错误信息。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// 调用 peer 的 Get 方法，尝试从远程节点获取指定组和 key 的缓存值。
	bytes, err := peer.Get(g.name, key)
	// 如果获取过程中出现错误，返回空的 ByteView 和错误信息。
	if err != nil {
		return ByteView{}, err
	}
	// 如果获取成功，将获取到的字节切片封装成 ByteView 并返回。
	return ByteView{b: bytes}, nil
}

// getLocally 从本地数据源获取缓存值。
// 该方法接收一个 key 作为参数，尝试从本地数据源获取与 key 对应的缓存值。
// 如果获取成功，将缓存值添加到本地缓存中，并返回一个 ByteView 类型的缓存值；
// 如果出现错误，返回一个空的 ByteView 和错误信息。
func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用 Group 的 getter 方法，从本地数据源获取与 key 对应的字节切片。
	bytes, err := g.getter.Get(key)
	// 如果获取过程中出现错误，返回空的 ByteView 和错误信息。
	if err != nil {
		return ByteView{}, err
	}
	// 克隆字节切片，避免数据被外部修改。
	value := ByteView{b: cloneBytes(bytes)}
	// 将获取到的缓存值添加到本地缓存中。
	g.populateCache(key, value)
	// 返回缓存值和 nil 错误信息。
	return value, nil
}

// populateCache 将指定的键值对添加到 Group 的主缓存中。
// 该方法接收一个 key 字符串和一个 ByteView 类型的 value 作为参数。
// 它调用 Group 的 mainCache 的 add 方法，将 key 和 value 添加到缓存中。
func (g *Group) populateCache(key string, value ByteView) {
	// 调用 mainCache 的 add 方法，将 key 和 value 添加到缓存中
	g.mainCache.add(key, value)
}
