package geecache

import (
	"fmt"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		// 可以对key进行处理，通过分离，我们可以写很多种Getter函数，让用户从不同的地方获取和处理信息
		return []byte(key), nil
	})
	result, _ := f.Get("meowrain")
	fmt.Println(result)
}
