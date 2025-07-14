package internal

import (
	"sync"
	"time"
)

type Cache struct {
	data map[string]cacheEntry
	sync.Mutex
}

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

func NewCache(interval time.Duration) *Cache {
	cache := &Cache{
		data: make(map[string]cacheEntry),
	}
	go cache.reapLoop(interval)
	return cache
}

func (c *Cache) Add() (key string, val []byte) {
	c.Lock()
	defer c.Unlock()

	c.data[key] = cacheEntry{
		createdAt: time.Now(),
		val:       val,
	}
	return key, val
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.Lock()
	defer c.Unlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	// Optionally check if the entry is expired
	if time.Since(entry.createdAt) > 24*time.Hour { // Example expiration time
		delete(c.data, key)
		return nil, false
	}

	return entry.val, true
}

func (c *Cache) reapLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.Lock()
			now := time.Now()
			for key, entry := range c.data {
				if now.Sub(entry.createdAt) > interval {
					delete(c.data, key)
				}

			}
			c.Unlock()
		}
	}
}
