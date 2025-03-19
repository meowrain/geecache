package geecache

type ByteView struct {
	b []byte // read only
	// use byets to support image,video,etc..
}

// 实现Len()方法,ByteView就能当做value传入lru中了
/*
type Value interface {
	Len() int
}
*/
// 使用值接受者，因为是只读的

// 返回内存占用大小
func (v ByteView) Len() int {
	return len(v.b)
}

// 因为是只读的，所以用ByteSlice()函数返回一个拷贝副本
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func (v ByteView) String() string {
	return string(v.b)
}
