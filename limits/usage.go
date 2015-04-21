// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package limits

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/codehack/go-relax"
	"net/http"
	"strconv"
)

// Usage monitors request usage limits to the service, resource or to specific
// route(s). It uses Container objects to implement the token-bucket (TB) algorithm.
// TB is useful for limiting number of requests and burstiness.
//
// Each client is assigned a (semi) unique key and given a bucket of tokens
// to spend per request. If a client consumes all its tokens, a response is
// sent with HTTP status 429-"Too Many Requests". At this time the client won't
// be allowed any more requests until a renewal period has passed. Repeated
// attempts while the timeout is in effect will simply reset the timer,
// prolonging the wait and dropping then new request.
//
// See also, https://en.wikipedia.org/wiki/Token_bucket
type Usage struct {
	// Container is an interface implemented by the bucket device.
	// The default container, MemBucket, is a memory-based container which stores
	// keys in an LRU cache. This container monitors a maximum number of keys,
	// and this value should be according to the system's available memory.
	// Defaults to a MemBucket container, with the values:
	//
	// 		maxKeys  = 1000 // number of keys to monitor.
	// 		capacity = 100  // total tokens per key.
	// 		fillrate = 1    // tokens renewed per minute per key.
	//
	// See also, MemBucket
	Container

	// Ration is the number of tokens to consume per request.
	// Defaults to 1.
	Ration int

	// Keygen is a function used to generate semi-unique ID's for each client.
	// The default function, MD5RequestKey, uses an MD5 hash on client address
	// and user agent, or the username of an authenticated client.
	Keygen func(relax.Context) string
}

// Run processes the filter. No info is passed.
func (f *Usage) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Container == nil {
		f.Container = NewMemBucket(1000, 100, 1)
	}
	if f.Keygen == nil {
		f.Keygen = MD5RequestKey
	}
	if f.Ration == 0 {
		f.Ration = 1
	}
	return func(ctx *relax.Context) {
		// Usage limits
		key := f.Keygen(*ctx)
		tokens, when, ok := f.Consume(key, f.Ration)
		if !ok {
			ctx.Header().Set("Retry-After", strconv.Itoa(when))
			http.Error(ctx, http.StatusText(relax.StatusTooManyRequests), relax.StatusTooManyRequests)
			return
		}
		ctx.Header().Set("RateLimit-Limit", strconv.Itoa(f.Capacity()))
		ctx.Header().Set("RateLimit-Remaining", strconv.Itoa(tokens))
		ctx.Header().Set("RateLimit-Reset", strconv.Itoa(when))

		next(ctx)
	}
}

// MD5RequestKey returns a key made from MD5 hash of Request.RemoteAddr and
// Request.UserAgent. But if the client has been authenticated, it will
// use the username as key.
func MD5RequestKey(c relax.Context) string {
	if c.Info.Contains("auth.user") {
		return "quota:" + c.Info.Get("auth.user")
	}
	h := md5.New()
	host, _ := SplitPort(c.Request.RemoteAddr)
	h.Write([]byte(host))
	h.Write([]byte(c.Request.UserAgent()))
	return "quota:" + hex.EncodeToString(h.Sum(nil))
}

// SplitPort splits an host:port address and returns the parts.
func SplitPort(addr string) (string, string) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:]
		}
	}
	return addr, ""
}
