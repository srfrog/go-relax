# Go-Relax [![GoDoc](https://pkg.go.dev/badge/github.com/srfrog/go-relax)](https://pkg.go.dev/github.com/srfrog/go-relax) [![Go Report Card](https://goreportcard.com/badge/github.com/srfrog/go-relax?svg=1)](https://goreportcard.com/report/github.com/srfrog/go-relax)

*Build fast and complete RESTful APIs in [Go](http://golang.org)*

*Go-Relax* aims to provide the tools to help developers build RESTful web services, and information needed to abide by [REST](https://en.wikipedia.org/wiki/REST) architectural constraints using correct [HTTP semantics](http://tools.ietf.org/html/rfc7231).

## Quick Start

Install using "go get":

	go get github.com/srfrog/go-relax

Then import from your source:

	import "github.com/srfrog/go-relax"

View [example_test.go](https://github.com/srfrog/go-relax/blob/master/example/example_test.go) for an extended example of basic usage and features.

Also, check the [wiki](https://github.com/srfrog/go-relax/wiki) for HowTo's and recipes.

## Features

- Helps build API's that follow the REST concept using ROA principles.
- Built-in support of HATEOAS constraint with Web Linking header tags.
- Follows REST "best practices", with inspiration from Heroku and GitHub.
- Works fine along with ``http.ServeMux`` or independently as ``http.Handler``
- Supports different media types, and **mixed** for requests and responses.
- It uses **JSON** media type by default, but also includes XML (needs import).
- The default routing engine uses **trie with regexp matching** for speed and flexibility.
- Comes with a complete set of filters to build a working API. _"Batteries included"_
- Uses ``sync.pool`` to efficiently use resources when under heavy load.

#### Included filters

- [x] Content - handles mixed request/response encodings, language preference, and versioning.
- [x] Basic authentication - to protect any resource with passwords.
- [x] CORS - Cross-Origin Resource Sharing, for remote client-server setups.
- [x] ETag - entity tagging with conditional requests for efficient caching.
- [x] GZip - Dynamic gzip content data compression, with ETag support.
- [x] Logging - custom logging with pre- and post- request event support.
- [x] Method override - GET/POST method override via HTTP header and query string.
- [x] Security - Various security practices for request handling.
- [x] Limits - request throttler, token-based rate limiter, and memory limits.

#### Upcoming filters

- [ ] JSON-API support.
- [ ] JSON-Schema for validating requests and responses.
- [ ] Collection-JSON support.

## Documentation

The full code documentation is located at GoDoc:

[https://pkg.go.dev/github.com/srfrog/go-relax](https://pkg.go.dev/github.com/srfrog/go-relax)

The source code is thoroughly commented, have a look.

## Hello World

This minimal example creates a new Relax service that handles a Hello resource.
```go
package main

import (
   "github.com/srfrog/go-relax"
)

type Hello string

func (h *Hello) Index(ctx *relax.Context) {
   ctx.Respond(h)
}

func main() {
   h := Hello("hello world!")
   svc := relax.NewService("http://api.example.com/")
   svc.Resource(&h)
   svc.Run()
}
```

**$ curl -i -X GET http://api.example.com/hello**

Response:

```
HTTP/1.1 200 OK
Content-Type: application/json;charset=utf-8
Link: </hello>; rel="self"
Link: </hello>; rel="index"
Request-Id: 61d430de-7bb6-4ff8-84da-aff6fe81c0d2
Server: Go-Relax/0.5.0
Date: Thu, 14 Aug 2014 06:20:48 GMT
Content-Length: 14

"hello world!"
```

## Credits

**Go-Relax** is Copyright (c) [CastleBytes](http://castlebytes.com).
Published under an [MIT License](https://raw.githubusercontent.com/srfrog/go-relax/master/LICENSE)
