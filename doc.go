// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package relax is a framework of pluggable components to build RESTful API's. It
provides a thin layer over ``net/http`` to serve resources, without imposing a rigid
structure. It is meant to be used along ``http.ServeMux``, but will work as a replacement
as it implements ``http.Handler``.

The framework is divided into components: Encoding, Filters, Routing, Hypermedia
and, Resources. These are the parts of a complete REST Service. All the components
are designed to be pluggable (replaced) through interfaces by external packages.
Relax provides enough built-in functionality to assemble a complete REST API.

The system is based on Resource Oriented Architecture (ROA), and had some inspiration
from Heroku's REST API.
*/
package relax

// Version is the version of this package.
const Version = "0.6.2"
