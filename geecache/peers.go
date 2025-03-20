package geecache

import "mikucache/geecache/geecachepb"

type PeerPicker interface {
	// 用于根据传入的key 选择相应的PeerGetter
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// 用于从对应group 查找缓存值 对应于HTTP客户端
	Get(in *geecachepb.Request, out *geecachepb.Response) error
}
