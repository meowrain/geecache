package geecache_test

import (
	"geecache"
	"testing"
)

func TestGetter(t *testing.T) {
	var f geecache.Getter = geecache.GetterFunc(
		func(key string) ([]byte, error) {
			return []byte(key), nil
		},
	)
	bytes, err := f.Get("key1")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(bytes))
}
