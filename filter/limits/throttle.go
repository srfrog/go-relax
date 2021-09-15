// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package limits

import (
	"net/http"
	"time"

	"github.com/srfrog/go-relax"
)

// Throttle allows to limit the rate of requests to a resource per specific time duration.
// It uses Go's channels to receive time tick updates. If a request is made before the channel
// is updated, the request is dropped with HTTP status code 429-"Too Many Requests".
type Throttle struct {
	// Request is the number of requests to allow per time duration.
	// Defaults to 100
	Requests int

	// Burst is the number of burst requests allowed before enforcing a time limit.
	// Defaults to 0
	Burst int

	// Per is the unit of time to quantize requests. This value is divided by the
	// value of Requests to get the time period to throttle.
	// Defaults to 1 second (time.Second)
	Per time.Duration
}

// Run processes the filter. No info is passed.
func (f *Throttle) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Requests == 0 {
		f.Requests = 100
	}
	if f.Per == 0 {
		f.Per = time.Second
	}

	limiter := f.process()
	return func(ctx *relax.Context) {
		select {
		case <-limiter:
			next(ctx)
		default:
			http.Error(ctx, http.StatusText(relax.StatusTooManyRequests), relax.StatusTooManyRequests)
			return
		}
	}
}

func (f *Throttle) process() chan time.Time {
	limiter := make(chan time.Time, f.Burst)
	go func() {
		for i := 0; i < f.Burst; i++ {
			limiter <- time.Now()
		}
		for t := range time.Tick(f.Per / time.Duration(f.Requests)) {
			limiter <- t
		}
	}()
	return limiter
}
