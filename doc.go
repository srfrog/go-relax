// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package relax is a framework of pluggable components to build RESTful API's. It
provides a thin layer over net/http to serve resources, without imposing a rigid
structure. It is meant to be used along http.ServeMux, but will work as a replacement
as it implements http.Handler.

The framework is divided into components: Encoding, Filters, Routing, Logging and Resources.
These are the parts of a complete REST Service. All the components are designed to be
pluggable (replaced) through interfaces by external packages. Relax provides
enough built-in functionality to assemble a complete REST API.
The system is based on Resource Oriented Architecture (ROA), and had some inspiration
from Heroku's REST API.

Encoding

Once a request enters service context, all responses are encoded according to the
assigned encoder. Relax includes support for JSON encoding. Other types of encoding
can be added by implementing the Encoder interface.

In common, all requests may send an HTTP Accept header with the form:

	Accept: application/vnd.relax+{encoding-type}; version=XX; lang=YY

Where encoding-type is the short notation of the encoding, for example, "json" or "xml".
version is an optional string to the content version; API version. lang is the
preferred language notation in ISO format, en_US, for example. version and lang are
optional but encoding-type is not. If the Accept header is not sent, then encoding
is assumed to be satisfied by the encoder's format, for example, "application/json".

Filters

Relax favors the use of filters over middleware to pre and post-process all requests.
Filters are function closures that are chained in FIFO order. At any time, a filter
can stop a request by returning before the next chained filter is called. The final
link is to the resource handler.

Filters are run at different times during a request, and in order: Service, Resource and, Route.
Service filters are run before resource filters, and resource filters before route filters.
This allows some granularity to filters.

Relax comes with filters that provide basic functionality needed by most REST API's.
Some included filters: CORS, method override, security, basic auth and content negotiation.
Adding filters is a matter of creating new objects that implement the Filter interface.
The position of the next() handler function is important to the effect of the particular
filter execution.

Routing

The routing system is modular and can be replaced by creating an object that implements
the Router interface. This interface defines two functions: one that adds routes and
another that finds a handle to a resource.

Relax's default router is trieRegexpRouter. It takes full routes, with HTTP method and path, and
inserts them in a trie that can use regular expressions to match individual path segments.

trieRegexpRouter's path segment expressions (PSE) are match strings that are pre-compiled as
regular expressions. PSE's provide a simple layer of security when accepting values from
the path. Each PSE is made out of a {type:varname} format, where type is the expected type
for a value and varname is the name to give the variable that matches the value.

	"{word:varname}" // matches any word; alphanumeric and underscore.

	"{uint:varname}" // matches an unsigned integer.

	"{int:varname}" // matches a signed integer.

	"{float:varname}" // matches a floating-point number in decimal notation.

	"{date:varname}" // matches a date in ISO 8601 format.

	"{geo:varname}" // matches a geo location as described in RFC 5870

	"{hex:varname}" // matches a hex number, with optional "0x" prefix.

	"{varname}" // catch-all; matches anything. it may overlap other matches.

	"*" // translated into "{wild}"

Some sample routes supported by trieRegexpRouter:

	GET /api/users/@{word:name}

	GET /api/users/{uint:id}/*

	POST /api/users/{uint:id}/profile

	DELETE /api/users/{date:from}/to/{date:to}

	GET /api/cities/{geo:location}

	PATCH /api/investments/\${float:dollars}/fund

Since PSE's are compiled to regexp, care must be taken to escape characters that
might break the compilation.

Logging

Relax provides a very simple logging system that is intended to be replaced by
something more robust. The foundation is laid for logging systems that support
event levels. An object must implement the Logger interface to enhance logging.

Logging itself is individual to each application and it's almost impossible to
build a system that can handle all cases. Many Go packages implement competent
logging systems that should fit the Logger interface.

The default logging system is a slight enhancement of the log package with colored
prefixes for each event level.

Resources

The main purpose of this framework is build REST API's that can serve resources.
In Relax, a resource is any object that implements the Resourcer interface.
Although it might seem a bit strict to require some specific handlers, they
reinforce that resources are handled in one or many ways. Your application is
not required to implement all handlers but at least let the client know such
handlers are not implemented. A resource creates a namespace where all operations
for that resource happen.

	type Locations struct{
		City string
		Country string
	}

	func (l *Locations) List (rw relax.ResponseWriter, re *relax.Request) {}
	func (l *Locations) Read (rw relax.ResponseWriter, re *relax.Request) {}
	func (l *Locations) Create (rw relax.ResponseWriter, re *relax.Request) {}
	func (l *Locations) Update (rw relax.ResponseWriter, re *relax.Request) {}
	func (l *Locations) Delete (rw relax.ResponseWriter, re *relax.Request) {}

	loc := &Locations{} // "locations" is our resource namespace.

	// CRUD will create routes to the handlers required by Resourcer using
	// "{geo:point}" as PSE.
	resource := service.Resource(loc).CRUD("geo:point")

*/
package relax

// Version is the version of this package.
const Version = "0.1.0"
