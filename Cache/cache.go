package cache

import (
	"sync"
)

type CacheEntry struct {
	URL   string `json:"url"`
	Count int    `json:"count"`
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

	return entry.URL, ok
}

func (c *Cache) Set(key string, value string) {

	entry := CacheEntry{
		URL:   value,
		Count: 1,
	}

	c.data.Store(key, entry)
}

func (c *Cache) Delete(key string) {
	c.data.Delete(key)
}

func (c *Cache) TopK(howmany int) []CacheEntry {

	var topK []CacheEntry
	count := 0
	c.data.Range(func(_, value any) bool {
		if count >= howmany {
			return false
		}

		entry := value.(CacheEntry)
		topK = append(topK, entry)
		count++
		return true // continue iteration
	})

	c.data.Range(func(_, value any) bool {

		entry := value.(CacheEntry)

		found := false
		for _, e := range topK {
			if e.URL == entry.URL {
				found = true
				break
			}
		}
		if found {
			return true
		}

		minIndex := 0
		for i := 1; i < len(topK); i++ {
			if topK[i].Count < topK[minIndex].Count {
				minIndex = i
			}
		}

		if topK[minIndex].Count < entry.Count {
			topK[minIndex] = entry
		}
		return true // continue iteration
	})

	return topK
}
