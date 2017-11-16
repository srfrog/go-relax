// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"context"
)

// serverVersion is used with the Server HTTP header.
const serverVersion = "Go-Relax/" + Version

// Logger interface is based on Go's ``log`` package. Objects that implement
// this interface can provide logging to Relax resources.
type Logger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})
}

// Service contains all the information about the service and resources handled.
// Specifically, the routing, encoding and service filters.
// Additionally, a Service is a collection of resources making it a resource by itself.
// Therefore, it implements the Resourcer interface. See: ``Service.Root``
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
	// uptime is a timestamp when service was started
	uptime time.Time
	// logger is the service logging system.
	logger Logger
	// Recovery is a handler function used to intervene after panic occur.
	Recovery http.HandlerFunc
}

// Logf prints an log entry to logger if set, or stdlog if nil.
// Based on the unexported function logf() in ``net/http``.
func (svc *Service) Logf(format string, args ...interface{}) {
	if svc.logger == nil {
		log.Printf(format, args...)
		return
	}
	svc.logger.Printf(format, args...)
}

// Index is a handler that responds with a list of all resources managed
// by the service. This is the default route to the base URI.
// With this function Service implements the Resourcer interface which is
// a resource of itself (the "root" resource).
// FIXME: this pukes under XML (maps of course).
func (svc *Service) Index(ctx *Context) {
	resources := make(map[string]string)
	for _, r := range svc.resources {
		resources[r.name] = r.Path(true)
		for _, l := range r.links {
			if l.Rel == "collection" {
				ctx.Header().Add("Link", l.String())
			}
		}
	}
	ctx.Respond(resources)
}

// BUG(TODO): Complete PATCH support - http://tools.ietf.org/html/rfc5789, http://tools.ietf.org/html/rfc6902

// Options implements the Optioner interface to handle OPTION requests for the root
// resource service.
func (svc *Service) Options(ctx *Context) {
	options := map[string]string{
		"base_href":          svc.URI.String(),
		"mediatype_template": Content.Mediatype + "+{subtype}; version={version}; lang={language}",
		"version_default":    Content.Version,
		"language_default":   Content.Language,
		"encoding_default":   svc.encoders["application/json"].Accept(),
	}
	ctx.Respond(options)
}

// InternalServerError responds with HTTP status code 500-"Internal Server Error".
// This function is the default service recovery handler.
func InternalServerError(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// dispatch tries to connect the request to a resource handler. If it can't find
// an appropriate handler it will return an HTTP error response.
func (svc *Service) dispatch(ctx *Context) {
	handler, err := svc.router.FindHandler(ctx.Request.Method, ctx.Request.URL.Path, &ctx.PathValues)
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
Adapter creates a new request context, sets default HTTP headers, creates the
link-chain of service filters, then passes the request to content negotiation.
Also, it uses a recovery function for panics, that responds with HTTP status
500-"Internal Server Error" and logs the event.

Info passed down by the adapter:

	ctx.Get("request.start_time").(time.Time)  // Time when request started, as string time.Time.
	ctx.Get("request.id").(string)             // Unique or user-supplied request ID.

Returns an http.HandlerFunc function that can be used with http.Handle.
*/
func (svc *Service) Adapter() http.HandlerFunc {
	handler := svc.dispatch
	for i := len(svc.filters) - 1; i >= 0; i-- {
		handler = svc.filters[i].Run(handler)
	}
	handler = svc.content(handler)

	// parent context
	parent := context.Background()

	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				svc.Recovery(w, r)
				svc.Logf("relax: Panic recovery: %s", err)
			}
		}()

		ctx := newContext(parent, w, r)
		defer ctx.free()

		requestID := NewRequestID(r.Header.Get("Request-Id"))

		ctx.Set("request.start_time", time.Now())
		ctx.Set("request.id", requestID)

		// set our default headers
		ctx.Header().Set("Server", serverVersion)
		ctx.Header().Set("Request-Id", requestID)

		handler(ctx)
	}
}

/*
Handler is a function that returns the values needed by http.Handle
to handle a path. This allows Relax services to work along http.ServeMux.
It returns the path of the service and the Service.Adapter handler.

	// restrict requests to host "api.codehack.com"
	myAPI := relax.NewService("http://api.codehack.com/v1")

	// ... your resources might go here ...

	// maps "api.codehack.com/v1" in http.ServeMux
	http.Handle(myAPI.Handler())

	// map other resources independently
	http.Handle("/docs", DocsHandler)
	http.Handle("/help", HelpHandler)
	http.Handle("/blog", BlogHandler)

	log.Fatal(http.ListenAndServe(":8000", nil))

Using this function with http.Handle is _recommended_ over using Service.Adapter
directly. You benefit from the security options built-in to http.ServeMux; like
restricting to specific hosts, clean paths, and separate path matching.
*/
func (svc *Service) Handler() (string, http.Handler) {
	if svc.URI.Host != "" {
		svc.Logf("relax: Matching requests to host %q", svc.URI.Host)
	}
	return svc.URI.Host + svc.URI.Path, svc.Adapter()
}

/*
ServeHTTP implements http.HandlerFunc. It lets the Service route all requests
directly, bypassing http.ServeMux.

	myService := relax.NewService("/")
	// ... your resources might go here ...

	// your service has complete handling of all the routes.
	log.Fatal(http.ListenAndServe(":8000", myService))

Using Service.Handler has more benefits than this method.
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

	myservice.Use(&cors.Filter{})
	myservice.Use(&security.Filter{CacheDisable: true})

To add encoders, assign an object that implements the Encoder interface.
Encoders will replace any matching existing encoder(s), and they will
be discoverable on the service encoders map.

	newenc := NewEncoderXML() // encoder with default settings
	newenc.Indented = true    // change a setting
	myservice.Use(newenc)     // assign it to service

To change the routing engine, assign an object that implements the
Router interface:

	myservice.Use(MyFastRouter())

To change the logging system, assign an object that implements the Logger
interface:

	// Use the excellent logrus package.
	myservice.Use(logrus.New())

	// With advanced usage
	log := &logrus.Logger{
		Out: os.Stderr,
		Formatter: new(JSONFormatter),
		Level: logrus.Debug,
	}
	myservice.Use(log)

Any entities that don't implement the required interfaces, will be ignored.
*/
func (svc *Service) Use(entities ...interface{}) *Service {
	for _, e := range entities {
		switch entity := e.(type) {
		case Encoder:
			svc.encoders[entity.Accept()] = entity
		case Filter:
			svc.filters = append(svc.filters, entity)
		case Router:
			svc.router = entity
		case Logger:
			svc.logger = entity
		default:
			svc.Logf("relax: Unknown entity to use: %T", entity)
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

// Logger returns the service logging system.
func (svc *Service) Logger() Logger {
	return svc.logger
}

// Uptime returns the service uptime in seconds.
func (svc *Service) Uptime() int {
	return int(time.Since(svc.uptime) / time.Second)
}

// Path returns the base path of this service.
// absolute whether or not it should return an absolute URL.
func (svc *Service) Path(absolute bool) string {
	path := svc.URI.Path
	if absolute {
		path = svc.URI.String()
	}
	return path
}

// Root points to the root resource, the service itself -- a collection of resources.
// This allows us to manipulate the service as a resource.
//
// Example:
//
//    // Create a new service mapped to "/v2"
//    svc := relax.NewService("/v2")
//
//    // Route /v2/status/{level} to SystemStatus() via root
//    svc.Root().GET("status/{word:level}", SystemStatus, &etag.Filter{})
//
// This is similar to:
//
//    svc.AddRoute("GET", "/v2/status/{level}", SystemStatus)
//
// Except that route-level filters can be used, without needing to meddle with
// service filters (which are global).
//
func (svc *Service) Root() *Resource {
	return svc.resources[0]
}

/*
Run will start the service using basic defaults or using arguments
supplied. If 'args' is nil, it will start the service on port 8000.
If 'args' is not nil, it expects in order: address (host:port),
certificate file and key file for TLS.

Run() is equivalent to:
	http.Handle(svc.Handler())
	http.ListenAndServe(":8000", nil)

Run(":3000") is equivalent to:
	...
	http.ListenAndServe(":3000", nil)

Run("10.1.1.100:10443", "tls/cert.pem", "tls/key.pem") is eq. to:
	...
	http.ListenAndServeTLS("10.1.1.100:10443", "tls/cert.pem", "tls/key.pem", nil)

If the key file is missing, TLS is not used.

*/
func (svc *Service) Run(args ...string) {
	var err error

	addr := ":8000"
	if args != nil {
		addr = args[0]
	}

	http.Handle(svc.Handler())

	if len(args) == 3 {
		svc.Logf("relax: Listening on %q (TLS)", addr)
		err = http.ListenAndServeTLS(addr, args[1], args[2], nil)
	} else {
		svc.Logf("relax: Listening on %q", addr)
		err = http.ListenAndServe(addr, nil)
	}

	if err != nil {
		log.Fatal(err)
	}
}

/*
NewService returns a new Service that can serve resources.

'uri' is the URI to this service, it should be an absolute URI but not required.
If an existing path is specified, the last path is used. 'entities' is an
optional value that contains a list of Filter, Encoder, Router objects that
are assigned at the service-level; the same as Service.Use().

	myservice := NewService("https://api.codehack.com/v1", &eTag.Filter{})

This function will panic if it can't parse 'uri'.
*/
func NewService(uri string, entities ...interface{}) *Service {
	u, err := url.Parse(uri)
	if err != nil {
		log.Panicln("relax: Service URI parsing failed:", err.Error())
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
		uptime:    time.Now(),
		Recovery:  InternalServerError,
	}

	// Make JSON the default encoder
	svc.Use(NewEncoder())
	// svc.encoders["application/json"] = NewEncoder()

	// Assign initial service entities
	if entities != nil {
		svc.Use(entities...)
	}

	// Setup the root resource
	root := &Resource{
		service:    svc,
		name:       "_root",
		path:       strings.TrimSuffix(u.Path, "/"),
		collection: svc,
	}

	// Default service routes
	root.Route("GET", "", svc.Index)
	root.Route("OPTIONS", "", root.OptionsHandler)

	svc.resources = append(svc.resources, root)

	log.Printf("relax: New service %q", u.String())

	return svc
}
