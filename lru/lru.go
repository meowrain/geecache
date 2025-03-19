package lru

import "container/list"

type Cache struct {
	maxBytes int64
	nbytes   int64
	ll       *list.List
	// 键是字符串，值是双向链表中对应节点的指针,list.Element就是entry结构体，entry表示一个节点，里面存储键值对
	cache map[string]*list.Element
	// 可选的，清除记录的时候调用
	OnEvicted func(key string, value Value)
}

// 双向链表节点的数据类型,链表中存储的都是这些节点，cache 中存储的是节点的指针
type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64, onEnvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		nbytes:    0,
		cache:     make(map[string]*list.Element),
		OnEvicted: onEnvicted,
	}
}

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		// 取出节点，更新值
		kv := ele.Value.(*entry)
		// key存在但是value不一样，那就需要更新已经用了的内存
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 如果不存在的话，lru算法就得把这个新节点加到头部了
		node := &entry{
			key:   key,
			value: value,
		}
		// PushFront会返回一个*list.Element 也就是会把*entry转换为*list.Element，存储在链表中
		ele := c.ll.PushFront(node)
		// 在map中记录
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	// 只要超过了最大内存，那就需要把老的节点删除掉
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}
func (c *Cache) Get(key string) (Value, bool) {
	if ele, ok := c.cache[key]; ok {
		// 如果这个值存在的话，那我们就把它放在队列头部
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return nil, false
}
func (c *Cache) RemoveOldest() {
	// 取链表尾部节点
	ele := c.ll.Back()
	if ele != nil {
		// 把节点从链表中删除
		kv := ele.Value.(*entry) // 从*list.Element转换为*entry,就可以提取key value，然后从cache 哈希表中删除key，和它对应的node指针，同时更新nbytes
		delete(c.cache, kv.key)
		c.ll.Remove(ele)
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
