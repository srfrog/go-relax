// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package limits

import (
	"time"

	"camlistore.org/pkg/lru"
)

// Container objects that implement this interface can serve as token bucket
// containers.
type Container interface {
	// Capacity returns the max number of tokens per client.
	Capacity() int

	// Consume takes tokens from a bucket.
	// Returns the number of tokens available, time in seconds for next one, and
	// a boolean indicating whether of not a token was consumed.
	Consume(string, int) (int, int, bool)

	// Reset will fill-up a bucket regardless of time/count.
	Reset(string)
}

// MemBucket implements Container using an in-memory LRU cache.
// This container is ideal for single-host applications, and it's go-routine
// safe.
type MemBucket struct {
	Size  int        // max tokens allowed, capacity.
	Rate  int        // tokens added per minute
	Cache *lru.Cache // LRU cache storage
}

type tokenBucket struct {
	Tokens int       // current token count
	When   time.Time // time of last check
}

// NewMemBucket returns a new MemBucket container object. It initializes
// the LRU cache with 'maxKeys'.
func NewMemBucket(maxKeys, capacity, rate int) Container {
	return &MemBucket{
		Size:  capacity,
		Rate:  rate,
		Cache: lru.New(maxKeys),
	}
}

// Capacity returns the total size of the container (bucket)
func (b *MemBucket) Capacity() int {
	return b.Size
}

// Consume removes a token from the key-indexed bucket at n-rate.
func (b *MemBucket) Consume(key string, n int) (int, int, bool) {
	tb := b.fill(key)
	if tb.Tokens < n {
		return tb.Tokens, b.wait(n - tb.Tokens), false
	}
	tb.Tokens -= n
	return tb.Tokens, b.wait(b.Size), true
}

// Reset re-fills the bucket and resets the rate.
func (b *MemBucket) Reset(key string) {
	cache, ok := b.Cache.Get(key)
	if ok {
		tb := cache.(*tokenBucket)
		tb.Tokens = b.Size
		tb.When = time.Now()
	}
}

func (b *MemBucket) wait(needed int) int {
	estimate := float64(needed/b.Rate) + float64(needed%b.Rate)*(1e-9/60.0)*60.0
	return int(estimate)
}

func (b *MemBucket) fill(key string) *tokenBucket {
	now := time.Now()
	cache, ok := b.Cache.Get(key)
	if !ok {
		tb := &tokenBucket{
			Tokens: b.Size,
			When:   now,
		}
		b.Cache.Add(key, tb)
		return tb
	}
	tb := cache.(*tokenBucket)
	if tb.Tokens < b.Size {
		delta := float64(b.Rate) * time.Since(tb.When).Minutes()
		tb.Tokens = Min(b.Size, tb.Tokens+int(delta))
	}
	tb.When = now
	return tb
}
