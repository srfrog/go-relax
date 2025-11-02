// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package relax

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
The position of the “next()“ handler function is important to the effect of the particular
filter execution.
*/
type Filter interface {
	// Run executes the current filter in a chain.
	// It takes a HandlerFunc function argument, which is executed within the
	// closure returned.
	Run(HandlerFunc) HandlerFunc
}

/*
LimitedFilter are filters that only can be used with a set of resources.
Where resource is one of: “Router“ (interface), “*Resource“ and “*Service“
The “RunIn()“ func should return true for the type(s) allowed, false otherwise.

	func (f *MyFilter) RunIn(r interface{}) bool {
		switch r.(type) {
		case relax.Router:
			return true
		case *relax.Resource:
			return true
		case *relax.Service:
			return false
		}
		return false
	}
*/
type LimitedFilter interface {
	RunIn(interface{}) bool
}
