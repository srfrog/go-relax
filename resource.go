// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"fmt"
	"net/http"
	"strings"
)

// Objects that implement the Resourcer interface will serve requests for a
// resource. A typical resource will implement one or all of the
// handlers in this interface, but those that aren't implemented should use
// the MethodNotAllowed() so expecting clients get a RESTful response.
type Resourcer interface {
	// Index may serve the entry GET request to a resource. Such as a listing of
	// resource items.
	Index(ResponseWriter, *Request)

	// Create may allow the creation of new resource items via methods POST/PUT.
	Create(ResponseWriter, *Request)

	// Read may display a specific resource item given an ID or name via method GET.
	Read(ResponseWriter, *Request)

	// Update may allow updating resource items via methods PATCH/PUT.
	Update(ResponseWriter, *Request)

	// Delete may allow removing items from a resource via method DELETE.
	Delete(ResponseWriter, *Request)
}

// Resource contains information about a resource. Resources are mapped under
// a Service.
type Resource struct {
	service    *Service    // service this resource belongs
	name       string      // name of this resource, derived from collection
	path       string      // path is the URI to this resource
	collection interface{} // the object that implements Resourcer; a collection
	filters    []Filter    // list of resource-level filters
	methods    string      // list of available methods
	links      []*Link     // resource links
}

// getPath similar as Service.getPath, returns the path to this resource. If sub
// is not empty, it appends to the resource path returned.
func (self *Resource) getPath(sub string) string {
	if strings.Contains(sub, self.path) {
		return sub
	}
	path := self.path
	if t := strings.Trim(sub, "/"); t != "" {
		path += "/" + t
	}
	return path
}

// optionsHandler responds to OPTION requests. It returns an Allow header listing
// the methods allowed for this resource.
// BUG(TODO): optionsHandler should peek for Authenticated routes.
func (self *Resource) optionsHandler(rw ResponseWriter, re *Request) {
	rw.Header().Set("Allow", self.methods)
	rw.WriteHeader(http.StatusNoContent)
}

// NotImplemented is a handler used to send a response when a resource route is
// not yet implemented.
func (self *Resource) NotImplemented(rw ResponseWriter, re *Request) {
	rw.Error(http.StatusNotImplemented, "That route is not implemented.")
}

// MethodNotAllowed is a handler used to send a response when a method is not
// allowed.
func (self *Resource) MethodNotAllowed(rw ResponseWriter, re *Request) {
	rw.Header().Set("Allow", self.methods)
	rw.Error(http.StatusMethodNotAllowed, "That method is not available for this resource.")
}

// relHandler is a resource filter that adds relations to the response.
func (self *Resource) relHandler(next HandlerFunc) HandlerFunc {
	return func(rw ResponseWriter, re *Request) {
		// FIXME: better relations here. this is a naive implementation.
		for _, link := range self.links {
			rw.Header().Add("Link", link.String())
		}
		next(rw, re)
	}
}

// Route adds a resource route (method + path) and its handler to the router.
// method is the HTTP method verb (GET, POST, ...).
// path is the URI path and optional matching expressions.
// h is the handler function with signature HandlerFunc (see Filter).
// filters are route-level filters run before the handler.
// Returns the resource itself for chaining.
//
// If the resource has its own filters, these are prepended to the filters list,
// resource-level filters will run before route-level filters.
func (self *Resource) Route(method, path string, h HandlerFunc, filters ...Filter) *Resource {
	handler := self.relHandler(h)
	if filters != nil {
		for i := len(filters) - 1; i >= 0; i-- {
			handler = filters[i].Run(handler)
		}
	}
	if self.filters != nil {
		for i := len(self.filters) - 1; i >= 0; i-- {
			handler = self.filters[i].Run(handler)
		}
	}
	// handler = self.Link(handler)
	method = strings.ToUpper(method)
	self.service.router.AddRoute(
		method,
		self.getPath(path),
		handler)

	// update methods list
	if !strings.Contains(self.methods, method) {
		if self.methods != "" {
			self.methods += ","
		}
		self.methods += method
	}

	return self
}

// DELETE is a convenient alias to Route using DELETE as method
func (self *Resource) DELETE(path string, h HandlerFunc, filters ...Filter) *Resource {
	return self.Route("DELETE", path, h, filters...)
}

// GET is a convenient alias to Route using GET as method
func (self *Resource) GET(path string, h HandlerFunc, filters ...Filter) *Resource {
	return self.Route("GET", path, h, filters...)
}

// OPTIONS is a convenient alias to Route using OPTIONS as method
func (self *Resource) OPTIONS(path string, h HandlerFunc, filters ...Filter) *Resource {
	return self.Route("OPTIONS", path, h, filters...)
}

// PATCH is a convenient alias to Route using PATCH as method
func (self *Resource) PATCH(path string, h HandlerFunc, filters ...Filter) *Resource {
	return self.Route("PATCH", path, h, filters...)
}

// POST is a convenient alias to Route using POST as method
func (self *Resource) POST(path string, h HandlerFunc, filters ...Filter) *Resource {
	return self.Route("POST", path, h, filters...)
}

// PUT is a convenient alias to Route using PUT as method
func (self *Resource) PUT(path string, h HandlerFunc, filters ...Filter) *Resource {
	return self.Route("PUT", path, h, filters...)
}

// CRUD creates Create/Read/Update/Delete routes using the handlers in Resourcer.
// pse is a route path segment expression (PSE).
// It returns the resource itself for chaining.
//
// For example, for a service under "/api/", given the Resourcer object "users",
// CRUD("{uint:id}") will add the following routes:
//
//		GET /api/users                => use handler users.Index()
//		GET /api/users/{uint:id}      => use handler users.Read()
//		POST /api/users               => use handler users.Create()
//		PUT /api/users                => Status: 405 Method not allowed
//		PUT /api/users/{uint:id}      => use handler users.Update()
//		DELETE /api/users             => Status: 405 Method not allowed
//		DELETE /api/users/{uint:id}   => use handler users.Delete()
//
// Other uses of PUT/PATCH/DELETE are dependent on the application, so CRUD()
// won't make any assumptions for those.
func (self *Resource) CRUD(pse string) *Resource {
	if pse == "" {
		// use resource collection name
		pse = "{" + strings.TrimRight(self.name, "s") + "}"
		if pse == "{}" {
			pse = "{item}" // give up
		}
	}

	self.Route("GET", "", (self.collection).(Resourcer).Index)
	self.Route("GET", pse, (self.collection).(Resourcer).Read)
	self.Route("POST", "", (self.collection).(Resourcer).Create)
	self.Route("PUT", "", self.MethodNotAllowed)
	self.Route("PUT", pse, (self.collection).(Resourcer).Update)
	self.Route("DELETE", "", self.MethodNotAllowed)
	self.Route("DELETE", pse, (self.collection).(Resourcer).Delete)

	return self
}

// Resource creates a new resource under Service that accepts REST requests.
// It will add an OPTIONS route that replies with an Allow header listing
// the methods available, along other default headers.
// This returns the new Resource object.
//
// collection is an object that implements the Resourcer interface.
// filters are resource-level filters that are ran before a resource handler, but
// after service-level filters.
//
// This function will panic if it can't determine the name of an collection
// through reflection.
func (svc *Service) Resource(collection Resourcer, filters ...Filter) *Resource {
	// reflect name from object's type
	cs := fmt.Sprintf("%T", collection)
	name := strings.ToLower(cs[strings.LastIndex(cs, ".")+1:])
	if name == "" {
		panic(`relax: Resource(` + cs + `): failed to reflect name of collection`)
	}

	res := &Resource{
		service:    svc,
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

	Log.Println(LOG_DEBUG, "New resource:", res.path, "=>", len(filters), "filters")

	// OPTIONS lists the methods allowed.
	res.Route("OPTIONS", "", res.optionsHandler)

	// update service resources list
	svc.resources = append(svc.resources, res)

	// Relation: resource -> service
	svc.links = append(svc.links, &Link{URI: res.path, Rel: svc.getPath("rel/"+name, true)})

	// Relation: index -> resource.path
	res.links = append(res.links, &Link{URI: res.path, Rel: "index"})

	return res
}
