package data

import (
	"fmt"

	"golift.io/cache"
)

// store provides a shared concurrency-safe data cache for our triggers (and web server).
// This cache is also immune from being purged during reload.
var store = cache.New(cache.Config{}) //nolint:gochecknoglobals

// Save a piece of data in the cache.
func Save(key string, data interface{}) {
	store.Save(key, data, cache.Options{})
}

// Get an itemfrom the cache. May be nil if non-existent.
func Get(key string) *cache.Item {
	return store.Get(key)
}

// SaveWithID saves data to the cache, and appends the key to an id.
func SaveWithID(key string, id int, data interface{}) {
	store.Save(key+fmt.Sprint(id), data, cache.Options{})
}

// GetWithID returns data from the cache using a kay appended to an id.
func GetWithID(key string, id int) *cache.Item {
	return store.Get(key + fmt.Sprint(id))
}
