// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"code.google.com/p/go-uuid/uuid"
	"net/http"
	"net/url"
	"time"
)

// Service contains all the information about the service and resources handled.
// Specifically, the routing, encoding and service filters.
type Service struct {
	// baseURI is the full reference URI to the service.
	baseURI string
	// path is the base path.
	path string
	// router is the routing engine
	router Router
	// encoders contains a list of our service media encoders.
	// Format: {mediatype}:{encoder object}. e.g., encoders["application/json"].
	encoders map[string]Encoder
	// filters are the service-level filters; which are run for all incoming requests.
	filters []Filter
	// resources is a list of all mapped resources
	resources []*Resource
	// links contains all the relation links
	links []*Link
}

// getPath returns the base path of this service.
// sub is a subpath segment to append to the path.
// absolute whether or not it should return an absolute path.
func (self *Service) getPath(sub string, absolute bool) string {
	path := self.path
	if absolute {
		path = self.baseURI
	}
	if sub != "" {
		path += sub
	}
	return path
}

// needsRequestId checks whether or not we need to create a new request id.
// this allows the main program to assign its own id's. id is the value
// from the Request-Id HTTP header.
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

/*
context creates a new context using managed ResponseWriter/Request, sets default
HTTP headers, log tracking and initiates the link-chain of filters ran before a
request is dispatched to a resource handler.

Additional request info:
	context.start_time:	Unix timestamp when request started
	context.client_addr:	IP address of request (best guess).
	context.request_id:	Unique or program-supplied request ID.
*/
func (self *Service) context(next HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r_start, r_addr, r_id := time.Now(), r.RemoteAddr, r.Header.Get("Request-Id")

		// check if the IP address is hidden behind a proxy request.
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			r_addr = ip
		} else if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			r_addr = ip
		}

		rw := newResponseWriter(w)
		re := newRequest(r)
		defer rw.free()
		defer re.free()

		if needsRequestId(r_id) {
			r_id = uuid.New()
		}

		// set our default headers
		rw.Header().Set("Server", "Go-Relax/"+Version)
		rw.Header().Set("Request-Id", r_id)
		rw.Header().Add(LinkHeader(re.URL.Path, `rel="self"`))

		// filter info
		re.Info.Set("context.start_time", r_start.Unix()) // request start time
		re.Info.Set("context.client_addr", r_addr)        // ip address
		re.Info.Set("context.request_id", r_id)           // request id

		Log.Printf(LOG_DEBUG, "%s method=%s uri=%s proto=%q addr=%s ua=%q", r_id, r.Method, r.URL.String(), r.Proto, r_addr, r.UserAgent())

		next(rw, re)

		Log.Printf(StatusLogLevel(rw.Status()), "[%.8s] \"%s %s\" => \"%d %s\" done in %fs", r_id, r.Method, r.URL.RequestURI(), rw.Status(), http.StatusText(rw.Status()), time.Since(r_start).Seconds())
	}
}

// dispatch tries to connect the request to a resource handler. If it can't find
// an appropiate handler it will return an HTTP error response.
func (self *Service) dispatch(rw ResponseWriter, re *Request) {
	handler, err := self.router.FindHandler(re)
	if err != nil {
		rw.Header().Set("Cache-Control", "max-age=300, stale-if-error=600")
		rw.Error(err.(*StatusError).Code, err.Error(), err.(*StatusError).Details)
		return
	}
	handler(rw, re)
}

// optionsHandler responds to OPTION requests. It returns an Allow header listing
// the methods allowed for this service path.
func (self *Service) optionsHandler(rw ResponseWriter, re *Request) {
	rw.Header().Set("Allow", "OPTIONS,GET")
	rw.WriteHeader(http.StatusNoContent)
}

// Map is a handler that responds with a list of all resources managed by the
// service. This is the default route to the baseURI.
// FIXME: Map needs to respond with a complete service map in JSON-LD.
func (self *Service) Map(rw ResponseWriter, re *Request) {
	var serviceMap struct {
		Resources map[string]string `json:"resources"`
		Media     struct {
			Type     string   `json:"type"`
			Version  string   `json:"version"`
			Language string   `json:"language"`
			Encoders []string `json:"encoders"`
		} `json:"media"`
	}
	serviceMap.Resources = make(map[string]string, 0)
	for _, v := range self.resources {
		serviceMap.Resources[v.name] = self.getPath(v.name, true)
	}
	serviceMap.Media.Type = contentMediaType
	serviceMap.Media.Version = re.Info.Get("content.version")
	serviceMap.Media.Language = re.Info.Get("content.language")
	for k, _ := range self.encoders {
		serviceMap.Media.Encoders = append(serviceMap.Media.Encoders, k)
	}
	for _, link := range self.links {
		rw.Header().Add("Link", link.String())
	}
	rw.Respond(serviceMap)
}

/*
Handler is a function that returns the parameters needed by http.Handle
to handle a path. This allows REST services to work along http.ServeMux.
It returns the path of the service and the context handler. The context
handler creates the managed request and response.

Info passed down from context:
	re.Info.Get("context.start_time") // Unix timestamp when request started
	re.Info.Get("context.client_addr) // IP address of request (best guess).
	re.Info.Get("context.request_id") // Unique or user-supplied request ID.
*/
func (self *Service) Handler() (string, http.Handler) {
	handler := self.dispatch
	for i := len(self.filters) - 1; i >= 0; i-- {
		handler = self.filters[i].Run(handler)
	}
	return self.path, self.context(handler)
}

/*
ServeHTTP lets the Service route all requests directly, bypassing
http.ServeMux.

It is recommended to use the Handler function with http.Handle
instead of this. But nothing should break if you do.
*/
func (self *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h := self.Handler()
	h.ServeHTTP(w, r)
}

/*
Use adds one or more encoders, filters and/or router to the service.
Returns the service itself, for chaining.

To add new filters, assign an object that implements the Filter interface.
Filters are not replaced or updated, only appended to the service list.
Examples:

	myservice.Use(&FilterCORS{})
	myservice.Use(&FilterSecurity{CacheDisable: true})

To add encoders, assign an object that implements the Encoder interface.
Encoders will replace any matching existing encoder(s), and they will
be discoverable on the service map.
Example:

	newenc := NewEncoderXML() // encoder with default settings
	newenc.Indented = true    // change a setting
	myservice.Use(newenc)     // assign it to service

To change the routing engine, assign an object that implements the
Router interface:

	myservice.Use(MyFastRouter())

Any entities that don't implement the require interfaces, will be ignored.
*/
func (self *Service) Use(entities ...interface{}) *Service {
	for _, e := range entities {
		switch entity := e.(type) {
		case Encoder:
			self.encoders[entity.Accept()] = entity
			Log.Printf(LOG_DEBUG, "Use encoder: %T", entity)
		case Filter:
			self.filters = append(self.filters, entity)
			Log.Printf(LOG_DEBUG, "Use filter: %T", entity)
		case Router:
			self.router = entity
			Log.Printf(LOG_DEBUG, "Use router: %T", entity)
		default:
			Log.Printf(LOG_NOTICE, "Unknown entity to use: %T", entity)
		}
	}
	return self
}

/*
Router returns the service routing engine.

The routing engine is responsible for creating routes (method + path)
to service resources, and accessing them for each request.
To add new routes you can use this interface directly:

	myservice.Router().AddRoute(method, path, handler)

Any route added directly with AddRoute() must reside under the service
URI base path, otherwise it won't work. No checks are made.
To find a handler to a request:

	h := myservice.Router().FindHandler(re)

This will return the handler associated for the route in the request 're'.
Where 're' is an *relax.Request object, usually sent through relax.HanderFunc.
*/
func (self *Service) Router() Router {
	return self.router
}

/*
NewService creates a new Service that can serve resources and returns it.

uri is the base URI to this service. It should be an absolute URI. If an
existing path is specified, the last path is used.

entities is an optional value that contains a list of Filter, Encoder, Router
objects that are assigned at the Service-level. This is the same as Service.Use.

	myservice := NewService("https://api.codehack.com/v1", &FilterETag{})

This function will panic if it can't parse the uri.
*/
func NewService(uri string, entities ...interface{}) *Service {
	url, err := url.Parse(uri)
	if err != nil {
		panic(err.Error())
	}

	if !url.IsAbs() {
		Log.Printf(LOG_WARN, "Service URI %q is not an absolute URI.", uri)
	}

	// the service path must end (and begin) with "/", this way ServeMux can
	// make a redirect for the non-absolute path.
	if url.Path == "" || url.Path[len(url.Path)-1] != '/' {
		url.Path += "/"
	}
	url.User = nil // XXX: should do something with this
	url.RawQuery = ""
	url.Fragment = ""

	svc := &Service{
		baseURI:   url.String(),
		path:      url.Path,
		router:    newRouter(),
		encoders:  make(map[string]Encoder),
		filters:   make([]Filter, 0),
		resources: make([]*Resource, 0),
		links:     make([]*Link, 0),
	}

	Log.Println(LOG_DEBUG, "New service:", svc.baseURI, "=>", svc.path)

	// Set the default encoder, EncoderJSON
	svc.encoders["application/json"] = NewEncoderJSON()

	// setup default service routes
	svc.router.AddRoute("GET", url.Path, svc.Map)
	svc.router.AddRoute("OPTIONS", url.Path, svc.optionsHandler)

	// The contentFilter, which provides content-negotiation, is the only default
	// service-level filter.
	svc.filters = append(svc.filters, &contentFilter{&svc.encoders})

	if entities != nil {
		svc.Use(entities...)
	}

	return svc
}
