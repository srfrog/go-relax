# Go-Relax [![GoDoc](https://godoc.org/github.com/codehack/go-relax?status.svg)](https://godoc.org/github.com/codehack/go-relax) ![Project progress](http://progressed.io/bar/45 "Progress")

*Build fast and complete RESTful APIs in [Go](http://golang.org)*

**Go-Relax** is a framework of pluggable components to build RESTful API's. It provides a thin layer over ``net/http`` to serve resources. It can be used along ``http.ServeMux``, but will work as a complement as it implements ``http.Handler``.

_Path to 1.0: Please keep note of the framework version as different areas are refactored and updated. Each commit is assumed stable but interfaces are not set in stone yet._

## Mission Statement

*Go-Relax* aims to provide the tools to help developers build RESTful web services, and information needed to abide by [REST](https://en.wikipedia.org/wiki/REST) architectural constraints using correct [HTTP semantics](http://tools.ietf.org/html/rfc7231).

## Features

- Helps build API's that follow the REST concept using ROA principles.
- Built-in support of HATEOAS constraint with Link header (and soon JSON-LD).
- It follows REST best practices, with inspiration from other REST API's like Heroku and GitHub's.
- Works fine along with ``http.ServeMux`` or independently as ``http.Handler``.
- Support for different media types, that can be **mixed** for requests and responses.
- It uses **JSON** media type by default, but also includes XML (not enabled by default).
- The default routing engine uses **trie with regexp matching** for speed and flexibility.
- Comes with a complete set of filters to build a working API. _"Batteries included"_
- All the framework's components: encoding, routing, logging, and filters, are modular. And should be easily replaced by custom packages.
- Uses ``sync.pool`` to efficiently use resources when under heavy load.

## Installation

Using "go get":

	go get github.com/codehack/go-relax

Then import from source:

	import "github.com/codehack/go-relax"

## Example

Check [example_test.go](https://github.com/codehack/go-relax/blob/master/example_test.go) for an example of basic usage.

## Howto

Howto's are found in the [wiki](https://github.com/codehack/go-relax/wiki).

## Documentation

The full code documentation is located at GoDoc:

[http://godoc.org/github.com/codehack/go-relax](http://godoc.org/github.com/codehack/go-relax)

**Go-Relax** is Copyright (c) 2014 [Codehack](http://codehack.com).
Published under [MIT License](https://raw.githubusercontent.com/codehack/go-relax/master/LICENSE)



