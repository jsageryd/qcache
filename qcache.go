// Package qcache implements a queue-based cache.
package qcache

import (
	"sync"
	"time"
)

// Cache is a key-value cache for arbitrary data.
type Cache struct {
	itemTTL          time.Duration
	items            map[any]*item
	maxPurgeInterval time.Duration
	mu               sync.RWMutex
	queue            []*item
	timer            *time.Timer
}

type item struct {
	expires time.Time
	key     any
	value   any
}

// New instantiates a new cache where items expire after given itemTTL.
func New(itemTTL time.Duration, options ...func(*Cache)) *Cache {
	c := &Cache{
		itemTTL:          itemTTL,
		items:            make(map[any]*item),
		maxPurgeInterval: 1 * time.Second,
		queue:            []*item{},
	}

	for _, o := range options {
		o(c)
	}

	return c
}

// WithMaxPurgeInterval limits the interval at which items are purged. It
// controls the balance of spending time to purge often in order to keep the
// cache at a more constant size, or purging less frequently at the expense of
// more variation in cache size. A max interval of 0 means each items is purged
// itemTTL after it has been added. A max interval of 10 seconds means expired
// items may stay in the cache for up to 10 seconds before they are removed. The
// default max interval is 1 second.
//
// This value does not affect the function of the Get method, as Get avoids
// returning expired items even if found in the cache, by checking the
// expiration time.
func WithMaxPurgeInterval(i time.Duration) func(*Cache) {
	return func(c *Cache) {
		if i >= 0 {
			c.maxPurgeInterval = i
		} else {
			c.maxPurgeInterval = 0
		}
	}
}

// ExpireAll expires all keys.
func (c *Cache) ExpireAll() {
	c.mu.Lock()
	c.items = make(map[any]*item)
	c.queue = []*item{}
	c.mu.Unlock()
}

// Get retreives the value for the given key.
func (c *Cache) Get(key any) (any, bool) {
	c.mu.RLock()

	if v, ok := c.items[key]; ok && !time.Now().After(v.expires) {
		c.mu.RUnlock()
		return v.value, ok
	}

	c.mu.RUnlock()

	return nil, false
}

// Set sets the given key to the given value. If the given key already exists,
// Set is a no-op.
func (c *Cache) Set(key any, value any) {
	if _, ok := c.Get(key); ok {
		return
	}

	c.mu.Lock()

	i := &item{
		expires: time.Now().Add(c.itemTTL),
		key:     key,
		value:   value,
	}

	c.items[key] = i
	c.queue = append(c.queue, i)

	if len(c.queue) == 1 {
		c.setTimer(c.itemTTL)
	}

	c.mu.Unlock()
}

// Size returns the number of items in the cache. Items expired but not yet
// purged are included.
func (c *Cache) Size() int {
	c.mu.RLock()
	s := len(c.items)
	c.mu.RUnlock()
	return s
}

func (c *Cache) setTimer(dur time.Duration) {
	if dur < c.maxPurgeInterval {
		dur = c.maxPurgeInterval
	}

	if c.timer == nil {
		c.timer = time.AfterFunc(dur, c.expire)
	} else {
		c.timer.Reset(dur)
	}
}

func (c *Cache) expire() {
	offset := 0
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	for offset < len(c.queue) && now.After(c.queue[offset].expires) {
		offset++
	}

	for _, item := range c.queue[:offset] {
		delete(c.items, item.key)
	}

	for i := 0; i < offset; i++ {
		c.queue[i] = nil
	}

	c.queue = c.queue[offset:]

	if len(c.queue) > 0 {
		dur := c.queue[0].expires.Sub(now)
		c.setTimer(dur)
	}
}
