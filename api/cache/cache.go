package cache

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgryski/go-farm"

	"video-chat/api/rtc"
)

const (
	DefaultCacheSize                = 1_000_000
	DefaultNumBuckets               = 100
	NoExpiration      time.Duration = -1
	DefaultExpiration time.Duration = 0
)

type Session = rtc.Session
type OnEvictHandler func(value *Session)

var (
	NotFoundError = errors.New("key not found")
	CloseError = errors.New("cache has been closed")
	MaxSizeExceed = errors.New("cache max size exceed")
)

type storedValue struct {
	object     *Session
	expiration int64
}

type Bucket struct {
	sync.RWMutex
	values map[uint64]storedValue
}

type evictEvent struct {
	bucketIdx int
	hash      uint64
}

//Cache typed cache
type Cache struct {
	cacheSize  int
	numBuckets int

	defaultExpiration time.Duration

	ticker     *time.Ticker
	evictionCh chan evictEvent

	size   int32
	closed int32

	storage []Bucket
	onEvict OnEvictHandler
}

func (item *storedValue) isExpired() bool {
	return item.expiration > 0 && time.Now().UnixNano() > item.expiration
}

//New typed cache constructor
func New(cacheSize int, numBuckets int, defaultExpiration, cleanupInterval time.Duration, onEvict OnEvictHandler) *Cache {
	if cacheSize <= 0 {
		cacheSize = DefaultCacheSize
	}
	if numBuckets <= 0 {
		numBuckets = DefaultNumBuckets
	}
	c := &Cache{
		cacheSize:         cacheSize,
		numBuckets:        numBuckets,
		defaultExpiration: defaultExpiration,
		ticker:            time.NewTicker(cleanupInterval),
		evictionCh:        make(chan evictEvent, cacheSize),
		onEvict:           onEvict,
	}
	c.storage = make([]Bucket, numBuckets)
	for i := 0; i < numBuckets; i++ {
		c.storage[i].values = make(map[uint64]storedValue)
	}
	go c.cacheEvict()
	return c
}

func (c *Cache) getBucketIdx(key string) (uint64, int) {
	hash := farm.Hash64([]byte(key))
	return hash, int(hash % uint64(c.numBuckets))
}

func (c *Cache) getExpiration(d time.Duration) int64 {
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		return time.Now().Add(d).UnixNano()
	}
	return 0
}

//Set value by key
func (c *Cache) Set(key string, value *Session, d time.Duration) (previous *Session, err error) {
	if c.isClosed() {
		err = CloseError
		return
	}
	h, bucketIdx := c.getBucketIdx(key)
	bucket := &c.storage[bucketIdx]
	bucket.Lock()
	item, ok := bucket.values[h]
	if ok {
		if !item.isExpired() {
			previous = item.object
		}
	} else {
		if c.cacheSize == c.getSize() {
			bucket.Unlock()
			err = MaxSizeExceed
			return
		}
		c.incSize()
	}
	bucket.values[h] = storedValue{value, c.getExpiration(d)}
	bucket.Unlock()
	return
}

func (c *Cache) Add(key string, value *Session, d time.Duration) (err error) {
	if c.isClosed() {
		err = CloseError
		return
	}
	if c.cacheSize == c.getSize() {
		err = MaxSizeExceed
		return
	}
	h, bucketIdx := c.getBucketIdx(key)
	bucket := &c.storage[bucketIdx]
	bucket.Lock()
	item, ok := bucket.values[h]
	if ok && !item.isExpired() {
		bucket.Unlock()
		err = errors.New("item '" + key + "' already exists")
		return
	}
	c.incSize()
	bucket.values[h] = storedValue{value, c.getExpiration(d)}
	bucket.Unlock()
	return
}

func (c *Cache) Replace(key string, value *Session, d time.Duration) (previous *Session, err error) {
	if c.isClosed() {
		err = CloseError
		return
	}
	h, bucketIdx := c.getBucketIdx(key)
	bucket := &c.storage[bucketIdx]
	bucket.Lock()
	item, ok := bucket.values[h]
	if !ok || item.isExpired() {
		bucket.Unlock()
		err = errors.New("item '" + key + "' doesn't exist")
		return
	}
	previous = item.object
	bucket.values[h] = storedValue{value, c.getExpiration(d)}
	bucket.Unlock()
	return
}

//Get value by key
func (c *Cache) Get(key string) (value *Session, err error) {
	if c.isClosed() {
		err = CloseError
		return
	}
	h, bucketIdx := c.getBucketIdx(key)
	bucket := &c.storage[bucketIdx]
	bucket.RLock()
	item, ok := bucket.values[h]
	bucket.RUnlock()
	if !ok || item.isExpired() {
		err = NotFoundError
		return
	}
	value = item.object
	return
}

//Delete value by key
func (c *Cache) Delete(key string) (err error) {
	if c.isClosed() {
		err = CloseError
		return
	}
	h, bucketIdx := c.getBucketIdx(key)
	bucket := &c.storage[bucketIdx]
	bucket.Lock()
	item, ok := bucket.values[h]
	if !ok {
		bucket.Unlock()
		err = NotFoundError
		return
	}
	if item.isExpired() {
		err = NotFoundError
	}
	c.decSize()
	delete(bucket.values, h)
	bucket.Unlock()
	return
}

func (c *Cache) cacheEvict() {
	bucketIdx := 0

	for !c.isClosed() {
		select {
		case <-c.ticker.C:
			bucket := &c.storage[bucketIdx]
			bucket.RLock()
			for hash, value := range bucket.values {
				if value.isExpired() {
					c.evictionCh <- evictEvent{bucketIdx, hash}
				}
			}
			bucket.RUnlock()
			if bucketIdx < c.numBuckets - 1 {
				bucketIdx++
			} else {
				bucketIdx = 0
			}
		case e := <-c.evictionCh:
			bucket := &c.storage[e.bucketIdx]
			bucket.Lock()
			item, ok := bucket.values[e.hash]
			if ok {
				c.decSize()
				delete(bucket.values, e.hash)
				if c.onEvict != nil {
					go c.onEvict(item.object)
				}
			}
			bucket.Unlock()
		}
	}
}

//Len get stored elements count
func (c *Cache) Size() int {
	if c.isClosed() {
		return 0
	}
	return c.getSize()
}

func (c *Cache) getSize() int {
	return int(atomic.LoadInt32(&c.size))
}

func (c *Cache) incSize() int {
	return int(atomic.AddInt32(&c.size, 1))
}

func (c *Cache) decSize() int {
	return int(atomic.AddInt32(&c.size, -1))
}

func (c *Cache) Close() {
	if c.isClosed() {
		return
	}
	atomic.StoreInt32(&c.closed, 1)
	c.ticker.Stop()
	close(c.evictionCh)
	for i := 0; i < c.numBuckets; i++ {
		bucket := &c.storage[i]
		bucket.Lock()
		bucket.values = make(map[uint64]storedValue)
		bucket.Unlock()
	}
	atomic.StoreInt32(&c.size, 0)
}

//Keep alive and return the element
func (c *Cache) KeepAlive(key string, d time.Duration) (value *Session, err error) {
	if c.isClosed() {
		err = CloseError
		return
	}
	h, bucketIdx := c.getBucketIdx(key)
	bucket := &c.storage[bucketIdx]
	bucket.Lock()
	item, ok := bucket.values[h]
	if !ok || item.isExpired() {
		bucket.Unlock()
		err = NotFoundError
		return
	}
	item.expiration = c.getExpiration(d)
	bucket.values[h] = item
	bucket.Unlock()
	value = item.object
	return
}

func (c *Cache) isClosed() bool {
	return atomic.LoadInt32(&c.closed) != 0
}
