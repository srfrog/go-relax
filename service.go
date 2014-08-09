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
	// URI is the full reference URI to the service.
	URI *url.URL
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
	// uptime is a timestamp when service was started
	uptime time.Time
}

// ServiceOptions has a description of the options available for using
// this service. This is used by the OPTIONS handler.
type ServiceOptions struct {
	BaseURI string `json:"href"`
	Media   struct {
		Type     string   `json:"type"`
		Version  string   `json:"version"`
		Language string   `json:"language"`
		Encoders []string `json:"encoders"`
	} `json:"media"`
}

// getPath returns the base path of this service.
// sub is a subpath segment to append to the path.
// absolute whether or not it should return an absolute path.
func (svc *Service) getPath(sub string, absolute bool) string {
	path := svc.URI.Path
	if absolute {
		path = svc.URI.String()
	}
	if sub != "" {
		path += sub
	}
	return path
}

// optionsHandler responds to OPTION requests. It reponds with the list of
// service options.
func (svc *Service) optionsHandler(ctx *Context) {
	ctx.Header().Set("Allow", svc.router.PathMethods(ctx.Request.URL.Path))
	ctx.Respond(svc.Options())
}

// rootHandler is a handler that responds with a list of all resources managed
// by the service. This is the default route to the baseURI.
// FIXME: this pukes under XML (maps of course).
func (svc *Service) rootHandler(ctx *Context) {
	resources := make(map[string]string, 0)
	for _, v := range svc.resources {
		resources[v.name] = svc.getPath(v.name, true)
	}
	for _, link := range svc.links {
		ctx.Header().Add("Link", link.String())
	}
	ctx.Respond(resources)
}

// NewRequestID returns a new request ID value based on UUID; or checks
// an id specified if it's valid for use as a request ID. If the id is not
// valid then it returns a new ID.
//
// A valid ID must be between 20 and 200 chars in length, and URL-encoded.
func NewRequestID(id string) string {
	if id == "" {
		return uuid.New()
	}
	l := 0
	for i, c := range id {
		switch {
		case 'A' <= c && c <= 'Z':
		case 'a' <= c && c <= 'z':
		case '0' <= c && c <= '9':
		case c == '-', c == '_', c == '.', c == '~', c == '%', c == '+':
		case i > 199:
			fallthrough
		default:
			return uuid.New()
		}
		l = i
	}
	if l < 20 {
		return uuid.New()
	}
	return id
}

// dispatch tries to connect the request to a resource handler. If it can't find
// an appropiate handler it will return an HTTP error response.
func (svc *Service) dispatch(ctx *Context) {
	handler, err := svc.router.FindHandler(ctx)
	if err != nil {
		ctx.Header().Set("Cache-Control", "max-age=300, stale-if-error=600")
		if err == ErrRouteBadMethod { // 405-Method Not Allowed
			ctx.Header().Set("Allow", svc.router.PathMethods(ctx.Request.URL.Path))
		}
		ctx.Error(err.(*StatusError).Code, err.Error(), err.(*StatusError).Details)
		return
	}
	handler(ctx)
}

/*
Adapter creates a new request context, sets default HTTP headers, log tracking,
creates the link-chain of service filters, then passes the request to content
negotiation.

Info passed down by the adapter:

	ctx.Info.Get("context.start_time")  // Unix timestamp when request started, in seconds.
	ctx.Info.Get("context.request_id")  // Unique or user-supplied request ID.

Returns an http.HandlerFunc function that can be used with http.Handle.
*/
func (svc *Service) Adapter() http.HandlerFunc {
	handler := svc.dispatch
	for i := len(svc.filters) - 1; i >= 0; i-- {
		handler = svc.filters[i].Run(handler)
	}
	handler = svc.Content(handler)

	return func(w http.ResponseWriter, r *http.Request) {
		when, ctx := time.Now(), NewContext(w, r)
		defer ctx.Free()

		requestID := NewRequestID(r.Header.Get("Request-Id"))

		ctx.Info.Set("context.start_time", when.Unix())
		ctx.Info.Set("context.request_id", requestID)

		// set our default headers
		ctx.Header().Set("Server", "Go-Relax/"+Version)
		ctx.Header().Set("Request-Id", requestID)
		ctx.Header().Add(LinkHeader(r.URL.Path, `rel="self"`))

		handler(ctx)

		Log.Printf(StatusLogLevel(ctx.Status()), "[%.8s] \"%s %s\" => \"%d %s\" done in %fs",
			requestID, r.Method, r.URL.RequestURI(), ctx.Status(), http.StatusText(ctx.Status()), time.Since(when).Seconds())
	}
}

/*
Handler is a function that returns the values needed by http.Handle
to handle a path. This allows Relax services to work along http.ServeMux.
It returns the path of the service and the Service.Adapter handler.

	myAPI := relax.NewService("http://codehack.com/api/v1")
	// ... your resources might go here ...

	// maps "/api/v1" in http.ServeMux
	http.Handle(myAPI.Handler())

	// map other resources independently
	http.Handle("/docs", DocsHandler)
	http.Handle("/help", HelpHandler)
	http.Handle("/blog", BlogHandler)

	log.Fatal(http.ListenAndServe(":8000", nil))

*/
func (svc *Service) Handler() (string, http.Handler) {
	return svc.URI.Path, svc.Adapter()
}

/*
ServeHTTP implements http.HandlerFunc. It lets the Service route all requests
directly, bypassing http.ServeMux.

	myService := relax.NewService("/")
	// ... your resources might go here ...

	// your service has complete handling of all the routes.
	log.Fatal(http.ListenAndServe(":8000", myService))

*/
func (svc *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	svc.Adapter().ServeHTTP(w, r)
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
be discoverable on the service encoders map.

	newenc := NewEncoderXML() // encoder with default settings
	newenc.Indented = true    // change a setting
	myservice.Use(newenc)     // assign it to service

To change the routing engine, assign an object that implements the
Router interface:

	myservice.Use(MyFastRouter())

Any entities that don't implement the require interfaces, will be ignored.
*/
func (svc *Service) Use(entities ...interface{}) *Service {
	for _, e := range entities {
		switch entity := e.(type) {
		case Encoder:
			svc.encoders[entity.Accept()] = entity
			Log.Printf(LogDebug, "Use encoder: %T", entity)
		case Filter:
			svc.filters = append(svc.filters, entity)
			Log.Printf(LogDebug, "Use filter: %T", entity)
		case Router:
			svc.router = entity
			Log.Printf(LogDebug, "Use router: %T", entity)
		default:
			Log.Printf(LogNotice, "Unknown entity to use: %T", entity)
		}
	}
	return svc
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

	h := myservice.Router().FindHandler(ctx)

This will return the handler for the route in request context 'ctx'.
*/
func (svc *Service) Router() Router {
	return svc.router
}

// Uptime returns the service uptime in seconds.
func (svc *Service) Uptime() int {
	return int(time.Since(svc.uptime) / time.Second)
}

// Options returns the options available from this service. This information
// is useful when creating OPTIONS routes.
func (svc *Service) Options() *ServiceOptions {
	options := &ServiceOptions{}
	options.BaseURI = svc.URI.String()
	options.Media.Type = ContentMediaType
	options.Media.Version = ContentDefaultVersion
	options.Media.Language = ContentDefaultLanguage
	for k := range svc.encoders {
		options.Media.Encoders = append(options.Media.Encoders, k)
	}
	return options
}

/*
NewService creates a new Service that can serve resources and returns it.

uri is the URI to this service, it should be an absolute URI but not required.
If an existing path is specified, the last path is used.

entities is an optional value that contains a list of Filter, Encoder, Router
objects that are assigned at the Service-level. This is the same as Service.Use.

	myservice := NewService("https://api.codehack.com/v1", &FilterETag{})

This function will panic if it can't parse the uri.
*/
func NewService(uri string, entities ...interface{}) *Service {
	u, err := url.Parse(uri)
	if err != nil {
		panic(err.Error())
	}

	if !u.IsAbs() {
		Log.Printf(LogWarn, "Service URI %q is not absolute.", uri)
	}

	// the service path must end (and begin) with "/", this way ServeMux can
	// make a redirect for the non-absolute path.
	if u.Path == "" || u.Path[len(u.Path)-1] != '/' {
		u.Path += "/"
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""

	svc := &Service{
		URI:       u,
		router:    newRouter(),
		encoders:  make(map[string]Encoder),
		filters:   make([]Filter, 0),
		resources: make([]*Resource, 0),
		links:     make([]*Link, 0),
		uptime:    time.Now(),
	}

	Log.Println(LogDebug, "New service:", u.String())

	// Set the default encoder, EncoderJSON
	svc.encoders["application/json"] = NewEncoderJSON()

	// setup default service routes
	svc.router.AddRoute("GET", u.Path, svc.rootHandler)
	svc.router.AddRoute("OPTIONS", u.Path, svc.optionsHandler)

	if entities != nil {
		svc.Use(entities...)
	}

	return svc
}
