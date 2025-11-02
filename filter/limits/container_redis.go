// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package limits

import (
	"net/url"
	"time"

	"github.com/garyburd/redigo/redis"
)

// RedisBucket implements Container using Redis strings.
type RedisBucket struct {
	Size int // max tokens allowed
	Rate int // tokens added per second
	Pool *redis.Pool
}

// Capacity returns the max number of tokens per client
func (b *RedisBucket) Capacity() int {
	return b.Size
}

// Consume takes tokens from a bucket.
// Returns the number of tokens available, time in seconds for next one, and
// a boolean indicating whether of not a token was consumed.
func (b *RedisBucket) Consume(key string, n int) (int, int, bool) {
	tokens := b.fill(key)
	if tokens < n {
		return tokens, b.wait(n - tokens), false
	}
	c := b.Pool.Get()
	defer c.Close()
	tokens, _ = redis.Int(c.Do("DECRBY", key, n))
	return tokens, b.wait(b.Size), true
}

// Reset will fill-up a bucket regardless of time/count.
func (b *RedisBucket) Reset(key string) {
	c := b.Pool.Get()
	defer c.Close()
	panicIf(c.Send("SET", key, b.Size, "EX", b.wait(b.Size), "XX"))
}

func (b *RedisBucket) wait(needed int) int {
	estimate := float64(needed/b.Rate) + float64(needed%b.Rate)*(1e-9/60.0)*60.0
	return int(estimate)
}

func (b *RedisBucket) fill(key string) int {
	var ttl, tokens int

	c := b.Pool.Get()
	defer c.Close()

	c.Send("MULTI")
	c.Send("TTL", key)
	c.Send("GET", key)
	values, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		c.Do("DISCARD")
		return 0
	}

	if _, err := redis.Scan(values, &ttl, &tokens); err != nil {
		panicIf(err)
		return 0
	}

	when := b.wait(b.Size)

	if ttl == -2 {
		panicIf(c.Send("SET", key, b.Size, "EX", when))
		return b.Size
	}

	if tokens < b.Size {
		since := when - ttl
		if since > 60 {
			delta := float64(b.Rate) * (time.Duration(since) * time.Second).Minutes()
			tokens = Min(b.Size, tokens+int(delta))
			panicIf(c.Send("SET", key, tokens, "EX", when, "XX"))
			return tokens
		}
	}

	panicIf(c.Send("EXPIRE", key, when))
	return tokens
}

// newRedisPool returns a new Redis connection pool.
// It expects an absolute URI with the format:
//
//	{network}://:{auth@}{host:port}/{index}
//
// Where:
//
//	{network} is "tcp" or "udp" for network type.
//		{auth} is authentication password.
//		{host:[port]} host address with optional port.
//		{index} an optional database index
//
// Example:
//
//	tcp://:secret@example.com:1234/5
//
// Defaults to port 6379 and index 0.
func newRedisPool(uri string) *redis.Pool {
	var auth, idx string

	u, err := url.Parse(uri)
	panicIf(err)

	if _, port := SplitPort(u.Host); port == "" {
		u.Host += ":6379"
	}

	if u.User != nil {
		if value, ok := u.User.Password(); ok {
			auth = value
		}
	}

	if u.Path != "" {
		idx = u.Path[1:]
	}

	return &redis.Pool{
		MaxIdle:     10,
		MaxActive:   100,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(u.Scheme, u.Host)
			if err != nil {
				return nil, err
			}
			if auth != "" {
				if err = c.Send("AUTH", auth); err != nil {
					c.Close()
					return nil, err
				}
			}
			if idx != "" {
				if err := c.Send("SELECT", idx); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

// NewRedisBucket returns a new Redis bucket.
func NewRedisBucket(uri string, capacity, rate int) *RedisBucket {
	return &RedisBucket{
		Size: capacity,
		Rate: rate,
		Pool: newRedisPool(uri),
	}
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
