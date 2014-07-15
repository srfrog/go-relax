// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"net/http"
	"reflect"
	"strings"
)

// Objects that implement the Resourcer interface will serve requests for a
// resource. A typical resource will implement one or all of the
// handlers in this interface, but those that aren't implemented should use
// the DefaultHandler() so expecting clients get a RESTful response.
type Resourcer interface {
	// List may serve the entry GET request to a resource. Such as a listing of
	// resource items.
	List(ResponseWriter, *Request)

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
	collection interface{} // the object that implements Resourcer
	filters    []Filter    // list of resource-level filters
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

// DefaultHandler is a handler used to send a RESTful response when a resource route is
// not yet implemented.
func (self *Resource) DefaultHandler(rw ResponseWriter, re *Request) {
	// BUG(TODO): DefaultHandler must add "Allow" header with methods which are allowed.
	rw.Error(http.StatusNotImplemented, "Resource path not handled.")
	// XXX: hmm... blame the client or the service?
	// rw.Error(http.StatusMethodNotAllowed, "resource path not handled.")
}

// Route adds a resource route (method + path) and its handler to the router. It returns
// the resource itself for chaining.
// method is the HTTP method verb (GET, POST, ...).
// path is the URI path and optional matching expressions.
// h is the handler function with signature HandlerFunc (see Filter).
// filters are route-level filters run before the handler.
//
// If the resource has its own filters, these are prepended to the filters list,
// resource-level filters will run before route-level filters.
func (self *Resource) Route(method, path string, h HandlerFunc, filters ...Filter) *Resource {
	handler := h
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
	self.service.router.AddRoute(
		strings.ToUpper(method),
		self.getPath(path),
		handler)
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
// itemid is a route patch matching expression (PSE) without {}'s.
// It returns the resource itself for chaining.
//
// For example, given the Resourcer object:
//		type Users struct{}
//
//	then, CRUD("uint:id") adds the following routes:
//		GET /api/users						=> users.List()
//		GET /api/users/{uint:id}		=> users.Read()
//		POST /api/users					=> users.Create()
//		PUT /api/users/{uint:id}		=> users.Update()
//		DELETE /api/users/{uint:id}	=> users.Delete()
func (self *Resource) CRUD(itemid string) *Resource {
	if itemid == "" {
		// detect a resource item type
		itemid = strings.TrimRight(self.name, "s")
		if itemid == "" {
			itemid = "itemid" // give up
		}
	}

	self.Route("GET", "", (self.collection).(Resourcer).List)
	self.Route("GET", "{"+itemid+"}", (self.collection).(Resourcer).Read)
	self.Route("POST", "", (self.collection).(Resourcer).Create)
	// self.Route("PUT", "", self.DefaultHandler)
	self.Route("PUT", "{"+itemid+"}", (self.collection).(Resourcer).Update)
	// self.Route("DELETE", "", self.DefaultHandler)
	self.Route("DELETE", "{"+itemid+"}", (self.collection).(Resourcer).Delete)
	// self.Route("OPTIONS", "", self.DefaultHandler)

	return self
}

// Resource creates a new resource under Service that accepts REST requests.
// collection is an object that implements the Resourcer interface.
// filters are resource-level filters that are ran before a resource handler, but
// after service-level filters.
// It returns the new Resource object.
//
// This function will panic if it can't determine the name of a collection
// through reflection.
func (s *Service) Resource(collection Resourcer, filters ...Filter) *Resource {
	// reflect name from object's definition
	cs := reflect.TypeOf(collection).String()
	name := strings.ToLower(cs[strings.LastIndex(cs, ".")+1:])
	if name == "" {
		panic(`relax: Resource(` + cs + `): failed to reflect name of collection`)
	}

	res := &Resource{s, name, s.getPath(name), collection, nil}

	// user-specified filters
	if filters != nil {
		// res.filters = make([]Filter)
		res.filters = append(res.filters, filters...)
	}

	Log.Println(LOG_DEBUG, "New resource:", name, "=>", len(filters), "filters")
	return res
}
