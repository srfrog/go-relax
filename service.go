// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"code.google.com/p/go-uuid/uuid"
	"net/http"
	"time"
)

// Service contains all the information about the service and resources handled.
// Specifically, the routing, encoding and service filters.
type Service struct {
	path    string   // path is the base URI path.
	router  Router   // router is the routing engine that routes URI paths to resource handlers.
	encoder Encoder  // encoder is the active encoding engine.
	filters []Filter // filters are the service-level filters; which are run for all incoming requests.
}

// getPath returns the base path of this service. If sub is not empty,
// it will append the value to the path.
func (self *Service) getPath(sub string) string {
	path := self.path
	if sub != "" {
		path += sub
	}
	return path
}

// needsRequestId checks whether or not we need to create a new request id.
// this allows the main program to assign its own id's. id is the value
// from the X-Request-ID HTTP header.
// Returns true if a new id is needed; false overwise.
func needsRequestId(id string) bool {
	if id == "" {
		return true
	}
	l := 0
	for i, c := range id {
		if i >= 200 {
			return true
		}
		switch {
		case '0' <= c && c <= '9':
			// continue
		case 'A' <= c && c <= 'Z':
			// continue
		case 'a' <= c && c <= 'z':
			// continue
		case '+' == c || '-' == c || '/' == c || '=' == c || '@' == c || '_' == c:
			// continue
		default:
			return true
		}
		l++
	}
	return l < 20
}

// context creates a new context using managed ResponseWriter/Request, sets default
// HTTP headers, log tracking and initiates the link-chain of filters ran before a
// request is dispatched to a resource handler.
//
// Additional request info:
//		context.start_time:	Unix timestamp when request started
//		context.client_addr:	IP address of request (best guess).
//		context.request_id:	Unique or program-supplied request ID.
func (self *Service) context(next HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r_start, r_addr, r_id := time.Now(), r.RemoteAddr, r.Header.Get("X-Request-ID")

		// check if the IP address is hidden behind a proxy request.
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			r_addr = ip
		} else if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			r_addr = ip
		}

		rw := &responseWriter{w: w, Encode: self.encoder.Encode}
		re := newRequest(r, &self.encoder)
		defer re.free()

		if needsRequestId(r_id) {
			r_id = uuid.New()
		}

		re.Info.Set("context.start_time", r_start.Unix()) // request start time
		re.Info.Set("context.client_addr", r_addr)        // ip address
		re.Info.Set("context.request_id", r_id)           // request id

		// set our default headers
		rw.Header().Set("Content-Type", self.encoder.ContentType())
		rw.Header().Set("X-Request-ID", r_id)
		rw.Header().Set("X-Powered-By", "Go Relax v"+Version)

		Log.Printf(LOG_INFO, "[%s] Request: %s %q for %s", r_id[:8], r.Method, r.URL.RequestURI(), r_addr)

		next(rw, re)

		Log.Printf(StatusLogLevel(rw.Status()), "[%s] Done: %d %q in %fs", r_id[:8], rw.Status(), http.StatusText(rw.Status()), time.Since(r_start).Seconds())
	}
}

// dispatch adapter tries to connect the request to a resource handler. If it can't find
// an appropiate handler it will return an HTTP error response.
func (self *Service) dispatch(rw ResponseWriter, re *Request) {
	handler, err := self.router.FindHandler(re)
	if err != nil {
		// XXX: not sure if we should be plain here.
		// http.Error(rw, err.Error(), err.(*StatusError).Code)
		rw.Error(err.(*StatusError).Code, err.Error(), err.(*StatusError).Details)
		return
	}
	handler(rw, re)
}

// Handler is a function that returns the parameters needed by http.Handle
// to handle a path. This allows REST services to work along http.ServeMux.
func (self *Service) Handler() (string, http.Handler) {
	handler := self.dispatch
	for i := len(self.filters) - 1; i >= 0; i-- {
		handler = self.filters[i].Run(handler)
	}
	return self.path, self.context(handler)
}

// ServeHTTP lets the Service route all requests directly, bypassing
// http.ServeMux.
//
// It is recommended to use the Handler function with http.Handle
// instead of this. But nothing should break if you do.
func (self *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h := self.Handler()
	h.ServeHTTP(w, r)
}

// Routing is used to change the routing engine.
// It expects a router object that implements the Router interface.
// Returns the service itself, for chaining.
func (self *Service) Routing(router Router) *Service {
	self.router = router
	return self
}

// Encoding is used to change the encoding engine. It expects an encoder object
// that implements the Encoder interface.
// Returns the service itself, for chaining.
func (self *Service) Encoding(encoder Encoder) *Service {
	self.encoder = encoder
	return self
}

// Filter adds new filter(s) to run at the Service level.
// Service-level filters will run for all requests, in the order they are assigned.
// Returns the service itself, for chaining.
func (self *Service) Filter(filter Filter) *Service {
	self.filters = append(self.filters, filter)
	return self
}

// NewService returns a new Service that can serve resources.
// path is the base path to this service. It should not overlap other service paths.
// If an existing path is specified, the last path is used.
// filters is an optional value that contains a list of Filter objects that
// run at the Service-level; which will run for all requests.
// The contentFilter, which provides content-negotiation, is the only default
// service-level filter.
// Returns the new service created.
func NewService(path string, filters ...Filter) *Service {
	// the service path must end (and begin) with "/", this way ServeMux can setup a redirect for the non-absolute path.
	if path == "" || path[len(path)-1] != '/' {
		path += "/"
	}

	svc := &Service{
		path:    path,
		router:  newRouter(),
		encoder: &EncoderJSON{},
		filters: make([]Filter, 0),
	}

	// setup default filters.
	svc.filters = append(svc.filters, &contentFilter{&svc.encoder})

	// user-specified filters
	if filters != nil {
		svc.filters = append(svc.filters, filters...)
	}

	return svc
}
