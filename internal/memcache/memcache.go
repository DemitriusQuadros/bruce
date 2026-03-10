package memcache

import (
	"sync"
)

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any)
	Delete(key string)
}

type inMemoryCache struct {
	data map[string]any
	mu   sync.RWMutex
}

func NewInMemoryCache() Cache {
	return &inMemoryCache{
		data: make(map[string]any),
	}
}

func (c *inMemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *inMemoryCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

func (c *inMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}
