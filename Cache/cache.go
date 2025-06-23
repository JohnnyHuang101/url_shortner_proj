package cache

import (
	"sort"
	"sync"
)

type CacheEntry struct {
	Original string `json:"original"`
	Hashed   string `json:"hashed"`
	Count    int    `json:"count"`
}

type Cache struct {
	data sync.Map
}

func NewCache() *Cache {
	return &Cache{}
}

func (c *Cache) Get(key string) (string, bool) {

	val, ok := c.data.Load(key)

	if !ok {
		return "", ok
	}

	entry := val.(CacheEntry)

	entry.Count++

	c.data.Store(key, entry)

	return entry.Original, ok
}

func (c *Cache) Set(key string, value string) {

	entry := CacheEntry{
		Original: value,
		Hashed:   "http://localhost:8080/r/" + key,
		Count:    1,
	}

	c.data.Store(key, entry)
}

func (c *Cache) Delete(key string) {
	c.data.Delete(key)
}

func (c *Cache) TopK(howmany int) []CacheEntry {
	var entries []CacheEntry

	// Collect all entries
	c.data.Range(func(_, value any) bool {
		entry := value.(CacheEntry)
		entries = append(entries, entry)
		return true
	})

	// Sort entries by Count in descending order
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	// Limit to top K
	if len(entries) > howmany {
		entries = entries[:howmany]
	}

	return entries
}
