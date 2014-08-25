// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package limits

import (
	"github.com/codehack/go-relax"
	"log"
	// "net/http"
	"time"
)

type Throttle struct {
	Container
	Requests int
	Per      time.Duration
}

// Run processes the filter. No info is passed.
// tooManyRequests responds with HTTP status 429-"Too Many Requests".
// A plain error is used in case of abuse.
// See https://tools.ietf.org/html/rfc6585#section-4
func (f *Throttle) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Container == nil {
		f.Container = NewMemBucket(1, 100, 1)
	}
	log.Printf("%#v", f)
	return func(ctx *relax.Context) {
		// Throttle requests.
		// Usage limits
		key := f.Keygen(*ctx)
		// println("KEY", key)
		tokens, when, ok := f.Consume(key, 1)
		if !ok {
			ctx.Header().Set("Retry-After", strconv.Itoa(when))
			http.Error(ctx, http.StatusText(relax.StatusTooManyRequests), relax.StatusTooManyRequests)
			return
		}
		ctx.Header().Set("RateLimit-Limit", strconv.Itoa(f.Capacity()))
		ctx.Header().Set("RateLimit-Remaining", strconv.Itoa(tokens))
		ctx.Header().Set("RateLimit-Reset", strconv.Itoa(when))

		next(ctx)
		<-c0.cycle
	}
}
