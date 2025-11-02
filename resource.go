// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package relax

import (
	"fmt"
	"net/http"
	"strings"
)

/*
Resourcer is any object that implements the this interface. A resource
is a namespace where all operations for that resource happen.

	type Locations struct{
		City string
		Country string
	}

	// This function is needed for Locations to implement Resourcer
	func (l *Locations) Index (ctx *Context) { ctx.Respond(l) }

	loc := &Locations{City: "Scottsdale", Country: "US"}
	myresource := service.Resource(loc)
*/
type Resourcer interface {
	// Index may serve the entry GET request to a resource. Such as the listing
	// of a collection.
	Index(*Context)
}

// Optioner is implemented by Resourcer objects that want to provide their own
// response to OPTIONS requests.
type Optioner interface {
	// Options may display details about the resource or how to access it.
	Options(*Context)
}

// CRUD is an interface for Resourcer objects that provide create, read,
// update, and delete operations; also known as CRUD.
type CRUD interface {
	// Create may allow the creation of new resource items via methods POST/PUT.
	Create(*Context)

	// Read may display a specific resource item given an ID or name via method GET.
	Read(*Context)

	// Update may allow updating resource items via methods PATCH/PUT.
	Update(*Context)

	// Delete may allow removing items from a resource via method DELETE.
	Delete(*Context)
}

// Resource is an object that implements Resourcer; serves requests for a resource.
type Resource struct {
	service    *Service    // service points to the service this resource belongs
	name       string      // name of this resource, derived from collection
	path       string      // path is the URI to this resource
	collection interface{} // the object that implements Resourcer; a collection
	links      []*Link     // links contains all the relation links
	filters    []Filter    // list of resource-level filters
}

// Path similar to Service.Path but returns the path to this resource.
// absolute whether or not it should return an absolute URL.
func (r *Resource) Path(absolute bool) string {
	return r.service.Path(absolute) + strings.TrimPrefix(r.path[len(r.service.Path(false))-1:], "/")
}

// NotImplemented is a handler used to send a response when a resource route is
// not yet implemented.
//
//	// Route "GET /myresource/apikey" => 501 Not Implemented
//	myresource.GET("apikey", myresource.NotImplemented)
func (r *Resource) NotImplemented(ctx *Context) {
	ctx.Error(http.StatusNotImplemented, "That route is not implemented.")
}

// MethodNotAllowed is a handler used to send a response when a method is not
// allowed.
//
//	// Route "PATCH /users/profile" => 405 Method Not Allowed
//	users.PATCH("profile", users.MethodNotAllowed)
func (r *Resource) MethodNotAllowed(ctx *Context) {
	ctx.Header().Set("Allow", r.service.router.PathMethods(ctx.Request.URL.Path))
	ctx.Error(http.StatusMethodNotAllowed, "The method "+ctx.Request.Method+" is not allowed.")
}

// OptionsHandler responds to OPTION requests. It returns an Allow header listing
// the methods allowed for an URI. If the URI is the Service's path then it returns information
// about the service.
func (r *Resource) OptionsHandler(ctx *Context) {
	methods := r.service.router.PathMethods(ctx.Request.URL.Path)
	ctx.Header().Set("Allow", methods)
	if strings.Contains(methods, "PATCH") {
		// FIXME: this is wrong! perhaps we need Patch.ContentType() or even Service.encoders keys.
		ctx.Header().Set("Accept-Patch", ctx.Get("content.encoding").(string))
	}
	if options, ok := r.collection.(Optioner); ok {
		options.Options(ctx)
		return
	}
	ctx.WriteHeader(http.StatusNoContent)
}

/*
Route adds a resource route (method + path) and its handler to the router.

'method' is the HTTP method verb (GET, POST, ...). 'path' is the URI path and
optional path matching expressions (PSE). 'h' is the handler function with
signature HandlerFunc. 'filters' are route-level filters run before the handler.
If the resource has its own filters, those are prepended to the filters list;
resource-level filters will run before route-level filters.

Returns the resource itself for chaining.
*/
func (r *Resource) Route(method, path string, h HandlerFunc, filters ...Filter) *Resource {
	handler := r.relationHandler(h)

	// route-specific filters
	handler = r.attachFilters(handler, filters...)

	// inherited resource filters
	handler = r.attachFilters(handler, r.filters...)

	r.service.router.AddRoute(strings.ToUpper(method), r.path+"/"+path, handler)

	return r
}

func (r *Resource) attachFilters(h HandlerFunc, filters ...Filter) HandlerFunc {
	if len(filters) == 0 {
		return h
	}
	for i := len(filters) - 1; i >= 0; i-- {
		if l, ok := filters[i].(LimitedFilter); ok && !l.RunIn(r.service.Router) {
			continue
		}
		h = filters[i].Run(h)
	}
	return h
}

// DELETE is a convenient alias to Route using DELETE as method
func (r *Resource) DELETE(path string, h HandlerFunc, filters ...Filter) *Resource {
	return r.Route("DELETE", path, h, filters...)
}

// GET is a convenient alias to Route using GET as method
func (r *Resource) GET(path string, h HandlerFunc, filters ...Filter) *Resource {
	return r.Route("GET", path, h, filters...)
}

// OPTIONS is a convenient alias to Route using OPTIONS as method
func (r *Resource) OPTIONS(path string, h HandlerFunc, filters ...Filter) *Resource {
	return r.Route("OPTIONS", path, h, filters...)
}

// PATCH is a convenient alias to Route using PATCH as method
func (r *Resource) PATCH(path string, h HandlerFunc, filters ...Filter) *Resource {
	return r.Route("PATCH", path, h, filters...)
}

// POST is a convenient alias to Route using POST as method
func (r *Resource) POST(path string, h HandlerFunc, filters ...Filter) *Resource {
	return r.Route("POST", path, h, filters...)
}

// PUT is a convenient alias to Route using PUT as method
func (r *Resource) PUT(path string, h HandlerFunc, filters ...Filter) *Resource {
	return r.Route("PUT", path, h, filters...)
}

/*
CRUD adds Create/Read/Update/Delete routes using the handlers in CRUD interface,
if the object implements it. A typical resource will implement one or all of the
handlers, but those that aren't implemented should respond with
"Method Not Allowed" or "Not Implemented".

pse is a route path segment expression (PSE) - see Router for details. If pse is
empty string "", then CRUD() will guess a value or use "{item}".

	type Jobs struct{}

	// functions needed for Jobs to implement CRUD.
	func (l *Jobs) Create (ctx *Context) {}
	func (l *Jobs) Read (ctx *Context) {}
	func (l *Jobs) Update (ctx *Context) {}
	func (l *Jobs) Delete (ctx *Context) {}

	// CRUD() will add routes handled using "{uint:ticketid}" as PSE.
	jobs := &Jobs{}
	myservice.Resource(jobs).CRUD("{uint:ticketid}")

The following routes are added:

	GET /api/jobs/{uint:ticketid}     => use handler jobs.Read()
	POST /api/jobs                    => use handler jobs.Create()
	PUT /api/jobs                     => Status: 405 Method not allowed
	PUT /api/jobs/{uint:ticketid}     => use handler jobs.Update()
	DELETE /api/jobs                  => Status: 405 Method not allowed
	DELETE /api/jobs/{uint:ticketid}  => use handler jobs.Delete()

Specific uses of PUT/PATCH/DELETE are dependent on the application, so CRUD()
won't make any assumptions for those.
*/
func (r *Resource) CRUD(pse string) *Resource {
	coll := r.collection.(CRUD)

	if pse == "" {
		// use resource collection name
		pse = "{" + strings.TrimRight(r.name, "s") + "}"
		if pse == "{}" {
			pse = "{item}" // give up
		}
	}

	r.Route("GET", pse, coll.Read)
	r.Route("POST", "", coll.Create)
	r.Route("PUT", "", r.MethodNotAllowed)
	r.Route("PUT", pse, coll.Update)
	r.Route("DELETE", "", r.MethodNotAllowed)
	r.Route("DELETE", pse, coll.Delete)

	r.NewLink(&Link{URI: r.Path(true) + "/" + pse, Rel: "item"})

	return r
}

/*
Resource creates a new Resource object within a Service, and returns it.
It will add an OPTIONS route that replies with an Allow header listing
the methods available. Also, it will create a GET route to the handler in
Resourcer.Index.

collection is an object that implements the Resourcer interface.

filters are resource-level filters that are ran before a resource handler, but
after service-level filters.

This function will panic if it can't determine the name of a collection
through reflection.
*/
func (svc *Service) Resource(collection Resourcer, filters ...Filter) *Resource {
	if collection == nil {
		panic("relax: Resource collection cannot be nil")
	}

	// check if the collection is the root resource
	cs := fmt.Sprintf("%T", collection)
	if cs == "*relax.Service" {
		return svc.Root()
	}

	// reflect name from object's type
	name := strings.ToLower(cs[strings.LastIndex(cs, ".")+1:])
	if name == "" {
		panic("relax: Resource naming failed: " + cs)
	}

	res := &Resource{
		service:    svc,
		name:       name,
		path:       svc.Path(false) + name,
		collection: collection,
		links:      make([]*Link, 0),
		filters:    nil,
	}

	// user-specified filters
	if len(filters) > 0 {
		for i := range filters {
			if l, ok := filters[i].(LimitedFilter); ok && !l.RunIn(res) {
				svc.Logf("relax: Filter not usable for resource: %T", filters[i])
				continue
			}
			res.filters = append(res.filters, filters[i])
		}
	}

	// OPTIONS lists the methods allowed.
	res.Route("OPTIONS", "", res.OptionsHandler)

	// GET on the collection will access the Index handler
	res.Route("GET", "", collection.Index)

	// Relation: index -> resource.path
	res.NewLink(&Link{URI: res.Path(true), Rel: svc.Path(true) + "rel/" + name})

	// Relation: resource -> service
	res.NewLink(&Link{URI: res.Path(true), Rel: "collection"})

	// update service resources list
	svc.resources = append(svc.resources, res)

	return res
}
