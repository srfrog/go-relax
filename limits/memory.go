// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package limits

import (
	"github.com/codehack/go-relax"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// Global memstats, shared by all Filter objects.
var c0 runtime.MemStats

// Memory sets limits on application and system memory usage. The memory
// stats are updated every minute and compared. If any limit is reached,
// a response is sent with HTTP status 503-"Service Unavailable".
// See also, runtime.MemStats
type Memory struct {
	// Allow sets a limit on current used memory size, in bytes. This value
	// ideally should be a number multiple of 2.
	// Defaults to 0 (disabled)
	// 	Alloc: 5242880 // 5MB
	Alloc uint64

	// Sys sets a limit on system memory usage size, in bytes. This value
	// ideally should be a number multiple of 2.
	// Defaults to 1e9 (1000000000 bytes)
	Sys uint64

	// RetryAfter is a suggested retry-after period, in seconds, as recommended
	// in http://tools.ietf.org/html/rfc7231#section-6.6.4
	// If zero, no header is sent.
	// Defaults to 0 (no header sent)
	RetryAfter int
}

// Run processes the filter. No info is passed.
func (f *Memory) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Sys == 0 {
		f.Sys = 1e9 // 1GB system memory
	}
	return func(ctx *relax.Context) {
		// Check memory limits
		if (f.Alloc != 0 && c0.Alloc > f.Alloc) || (f.Sys != 0 && c0.Sys > f.Sys) {
			if f.RetryAfter != 0 {
				ctx.Header().Set("Retry-After", strconv.Itoa(f.RetryAfter))
			}
			http.Error(ctx, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}

		next(ctx)
	}
}

// updateMemStats will update our MemStats values every minute.
func updateMemStats() {
	for _ = range time.Tick(time.Minute) {
		runtime.MemProfileRate = 0
		runtime.ReadMemStats(&c0)
	}
}

func init() {
	runtime.MemProfileRate = 0
	runtime.ReadMemStats(&c0)
	go updateMemStats()
}
