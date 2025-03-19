package lru

import (
	"testing"
)

type String string

func (s String) Len() int {
	return len(s)
}

func TestGet(t *testing.T) {
	lru := New(int64(20), nil)
	lru.Add("key1", String("1234"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	t.Log("lru len: ", lru.Len()) // 链表节点数
}

func TestGetMore(t *testing.T) {
	lru := New(10, nil)
	lru.Add("key1", String("1234"))  // 8
	lru.Add("key2", String("12345")) // 9
	// out of range ! 能拿到key2 但是拿不到key1，因为key1已经被移除了
	if _, ok := lru.Get("key2"); !ok {
		t.Fatalf("cache hit key2=12345 failed")
	}

	if _, ok := lru.Get("key1"); !ok {
		t.Fatalf("cache hit key1=1234 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}
	lru := New(10, callback)
	lru.Add("key1", String("123456")) // 10
	lru.Add("key2", String("123456")) // 10
	lru.Add("key3", String("123456")) // 10
	// total 30 > 10 onEvicted triger for 2 times
	expect := []string{"key1", "key2"}
	if len(keys) != len(expect) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
	t.Log("keys:", keys)
}
