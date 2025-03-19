package geecache

// 缓存不存在的时候，调用这个接口，获取源数据
type Getter interface {
	Get(key string) ([]byte, error)
}
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}
