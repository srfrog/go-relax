// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

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
	func (l *Locations) Index (ctx *Context) {}

	loc := &Locations{City: "Scottsdale", Country: "US"}
	myresource := service.Resource(loc)

*/
type Resourcer interface {
	// Index may serve the entry GET request to a resource. Such as a listing
	// a collection.
	Index(*Context)
}

// The CRUD interface is for Resourcer objects that provide create, read,
// update and delete operations, also known as CRUD.
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
	router     Router      // router used to map routes to resource handlers
	name       string      // name of this resource, derived from collection
	path       string      // path is the URI to this resource
	collection interface{} // the object that implements Resourcer; a collection
	filters    []Filter    // list of resource-level filters
	links      []*Link     // resource links
}

// getPath similar as Service.getPath, returns the path to this resource. If sub
// is not empty, it appends to the resource path returned.
func (r *Resource) getPath(sub string) string {
	if strings.Contains(sub, r.path) {
		return sub
	}
	path := r.path
	if t := strings.Trim(sub, "/"); t != "" {
		path += "/" + t
	}
	return path
}

// optionsHandler responds to OPTION requests. It returns an Allow header listing
// the methods allowed for this resource.
func (r *Resource) optionsHandler(ctx *Context) {
	ctx.Header().Set("Allow", r.router.PathMethods(ctx.Request.URL.Path))
	ctx.WriteHeader(http.StatusNoContent)
}

// NotImplemented is a handler used to send a response when a resource route is
// not yet implemented.
//		// Route "GET /myresource/apikey" => 501 Not Implemented
//		myresource.GET("apikey", myresource.NotImplemented)
func (r *Resource) NotImplemented(ctx *Context) {
	ctx.Error(http.StatusNotImplemented, "That route is not implemented.")
}

// MethodNotAllowed is a handler used to send a response when a method is not
// allowed.
//		// Route "PATCH /users/profile" => 405 Method Not Allowed
//		users.PATCH("profile", users.MethodNotAllowed)
func (r *Resource) MethodNotAllowed(ctx *Context) {
	ctx.Header().Set("Allow", r.router.PathMethods(ctx.Request.URL.Path))
	ctx.Error(http.StatusMethodNotAllowed, "The method "+ctx.Request.Method+" is not allowed.")
}

// relHandler is a resource filter that adds relations to the response.
func (r *Resource) relHandler(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		// FIXME: better relations here. this is a naive implementation.
		for _, link := range r.links {
			ctx.Header().Add("Link", link.String())
		}
		next(ctx)
	}
}

/*
Route adds a resource route (method + path) and its handler to the router.

method is the HTTP method verb (GET, POST, ...).

path is the URI path and optional matching expressions.

h is the handler function with signature HandlerFunc (see Filter).

filters are route-level filters run before the handler. If the resource has
its own filters, those are prepended to the filters list; resource-level
filters will run before route-level filters.

Returns the resource itself for chaining.
*/
func (r *Resource) Route(method, path string, h HandlerFunc, filters ...Filter) *Resource {
	handler := r.relHandler(h)
	if filters != nil {
		for i := len(filters) - 1; i >= 0; i-- {
			handler = filters[i].Run(handler)
		}
	}
	if r.filters != nil {
		for i := len(r.filters) - 1; i >= 0; i-- {
			handler = r.filters[i].Run(handler)
		}
	}
	r.router.AddRoute(strings.ToUpper(method), r.getPath(path), handler)

	return r
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

// BUG(TODO): Complete PATCH support - http://tools.ietf.org/html/rfc5789

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
	myservice.Resource(&Jobs{}).CRUD("{uint:ticketid}")

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
	crud, ok := r.collection.(CRUD)
	if !ok {
		Log.Printf(LogErr, "%T doesn't implement CRUD", r.collection)
		return r
	}

	if pse == "" {
		// use resource collection name
		pse = "{" + strings.TrimRight(r.name, "s") + "}"
		if pse == "{}" {
			pse = "{item}" // give up
		}
	}

	Log.Println(LogDebug, "Adding CRUD routes...")

	r.Route("GET", pse, crud.Read)
	r.Route("POST", "", crud.Create)
	r.Route("PUT", "", r.MethodNotAllowed)
	r.Route("PUT", pse, crud.Update)
	r.Route("DELETE", "", r.MethodNotAllowed)
	r.Route("DELETE", pse, crud.Delete)

	r.links = append(r.links, &Link{URI: r.getPath("{item}"), Rel: "edit"})

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
	// reflect name from object's type
	cs := fmt.Sprintf("%T", collection)
	name := strings.ToLower(cs[strings.LastIndex(cs, ".")+1:])
	if name == "" {
		panic(`relax: Resource(` + cs + `): failed to reflect name of collection`)
	}

	res := &Resource{
		router:     svc.router,
		name:       name,
		path:       svc.getPath(name, false),
		collection: collection,
		filters:    nil,
		links:      make([]*Link, 0),
	}

	// user-specified filters
	if filters != nil {
		res.filters = append(res.filters, filters...)
	}

	Log.Println(LogDebug, "New resource:", res.path, "=>", len(filters), "filters")

	// OPTIONS lists the methods allowed.
	res.Route("OPTIONS", "", res.optionsHandler)

	// GET on the collection will access the Index handler
	res.Route("GET", "", collection.Index)

	// Relation: index -> resource.path
	res.links = append(res.links, &Link{URI: svc.getPath(name, false), Rel: "index"})

	// update service resources list
	svc.resources = append(svc.resources, res)

	// Relation: resource -> service
	svc.links = append(svc.links, &Link{URI: svc.getPath(name, true), Rel: svc.getPath("rel/"+name, true)})

	return res
}
