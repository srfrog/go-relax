// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package limits_test

import (
	"time"

	"github.com/srfrog/go-relax"
	"github.com/srfrog/go-relax/filter/limits"
)

type Count int

func (c *Count) Index(ctx *relax.Context) {
	*c += 1
	ctx.Respond(c)
}

// Example_basic creates a new service under path "/" and serves requests
// for the count resource.
func Example_basic() {
	c := Count(0)
	svc := relax.NewService("/")

	// Memory limit check, allocation 250kb
	svc.Use(&limits.Memory{Alloc: 250 * 1024})

	// Throttle limit, 1 request per 200ms
	svc.Use(&limits.Throttle{
		Burst:    5,
		Requests: 1,
		Per:      time.Minute * 3,
	})

	// Usage limit check, 10 tokens
	svc.Use(&limits.Usage{
		Container: limits.NewRedisBucket("tcp://127.0.0.1", 10, 1),
	})

	svc.Resource(&c)
	svc.Run()
	// Output:
}
