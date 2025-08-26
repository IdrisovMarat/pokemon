package main

import (
	"sync"
)

// MockCache для тестирования
type MockCache struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string][]byte),
	}
}

func (c *MockCache) Add(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = val
}

func (c *MockCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

func (c *MockCache) Stop() {
	// Ничего не делаем для mock
}
