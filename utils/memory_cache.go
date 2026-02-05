package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value      interface{} `json:"value"`
	Expiration time.Time   `json:"expiration"`
}

// MemoryCache provides in-memory caching with expiration and disk persistence
type MemoryCache struct {
	items    map[string]*CacheItem
	mu       sync.RWMutex
	diskPath string
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(diskPath string) *MemoryCache {
	if diskPath != "" {
		os.MkdirAll(diskPath, 0755)
	}
	cache := &MemoryCache{
		items:    make(map[string]*CacheItem),
		diskPath: diskPath,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Set stores a value in cache with expiration
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := time.Now().Add(ttl)
	item := &CacheItem{
		Value:      value,
		Expiration: expiration,
	}
	c.items[key] = item
	
	if c.diskPath != "" {
		c.saveToDisk(key, item)
	}
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	item, exists := c.items[key]
	c.mu.RUnlock()

	if exists {
		// Check expiration
		if time.Now().After(item.Expiration) {
			c.Delete(key)
			return nil, false
		}
		return item.Value, true
	}
	
	// Fallback to disk if configured
	if c.diskPath != "" {
		if diskItem, found := c.loadFromDisk(key); found {
			if time.Now().After(diskItem.Expiration) {
				c.deleteFromDisk(key)
				return nil, false
			}
			// Restore to memory
			c.mu.Lock()
			c.items[key] = diskItem
			c.mu.Unlock()
			return diskItem.Value, true
		}
	}

	return nil, false
}

// Delete removes an item from cache
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	
	if c.diskPath != "" {
		c.deleteFromDisk(key)
	}
}

// Clear removes all items from cache
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]*CacheItem)
	c.mu.Unlock()
	
	// TODO: Clear disk items?
}

// cleanupLoop periodically removes expired items
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired items
func (c *MemoryCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.Expiration) {
			delete(c.items, key)
			if c.diskPath != "" {
				c.deleteFromDisk(key) // Async?
			}
		}
	}
}

// Size returns the number of items in cache
func (c *MemoryCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Has checks if a key exists in cache
func (c *MemoryCache) Has(key string) bool {
	_, exists := c.Get(key)
	return exists
}

// Keys returns all keys in cache
func (c *MemoryCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// Persistence helpers

func (c *MemoryCache) saveToDisk(key string, item *CacheItem) {
	filePath := filepath.Join(c.diskPath, key+".cache")
	file, err := os.Create(filePath)
	if err == nil {
		defer file.Close()
		json.NewEncoder(file).Encode(item)
	}
}

func (c *MemoryCache) loadFromDisk(key string) (*CacheItem, bool) {
	filePath := filepath.Join(c.diskPath, key+".cache")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, false
	}
	defer file.Close()

	var item CacheItem
	if err := json.NewDecoder(file).Decode(&item); err != nil {
		return nil, false
	}
	return &item, true
}

func (c *MemoryCache) deleteFromDisk(key string) {
	filePath := filepath.Join(c.diskPath, key+".cache")
	os.Remove(filePath)
}

// Global cache instance
var GlobalCache = NewMemoryCache("./cache/data")
