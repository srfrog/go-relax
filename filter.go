// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

// HandlerFunc is simply a version of http.HandlerFunc that uses Context.
// All filters must return and accept this type.
type HandlerFunc func(*Context)

/*
Filter is a function closure that is chained in FILO (First-In Last-Out) order.
Filters pre and post process all requests. At any time, a filter can stop a request by
returning before the next chained filter is called. The final link points to the
resource handler.

Filters are run at different times during a request, and in order: Service, Resource and, Route.
Service filters are run before resource filters, and resource filters before route filters.
This allows some granularity to filters.

Relax comes with filters that provide basic functionality needed by most REST API's.
Some included filters: CORS, method override, security, basic auth and content negotiation.
Adding filters is a matter of creating new objects that implement the Filter interface.
The position of the ``next()`` handler function is important to the effect of the particular
filter execution.
*/
type Filter interface {
	// Run executes the current filter in a chain.
	// It takes a HandlerFunc function argument, which is executed within the
	// closure returned.
	Run(HandlerFunc) HandlerFunc
}
