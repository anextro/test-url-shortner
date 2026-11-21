package main

import (
	"container/list"
	"sync"
)

// LRU is a fixed-size, mutex-guarded LRU cache of code -> url.
type LRU struct {
	mu    sync.Mutex
	max   int
	ll    *list.List
	items map[string]*list.Element
}

type entry struct{ key, val string }

func NewLRU(max int) *LRU {
	return &LRU{max: max, ll: list.New(), items: make(map[string]*list.Element)}
}

func (c *LRU) Get(k string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[k]; ok {
		c.ll.MoveToFront(e)
		return e.Value.(*entry).val, true
	}
	return "", false
}

func (c *LRU) Add(k, v string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[k]; ok {
		e.Value.(*entry).val = v
		c.ll.MoveToFront(e)
		return
	}
	c.items[k] = c.ll.PushFront(&entry{k, v})
	if c.ll.Len() > c.max {
		old := c.ll.Back()
		c.ll.Remove(old)
		delete(c.items, old.Value.(*entry).key)
	}
}

func (c *LRU) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}
