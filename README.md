# Go-Relax [![GoDoc](https://godoc.org/github.com/codehack/go-relax?status.svg)](https://godoc.org/github.com/codehack/go-relax) ![Project progress](http://progressed.io/bar/35 "Progress")

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
- Comes with a complete set of filters to build a working API. aka _"Batteries included"_ but not the kitchen sink.
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

This and more Howto's are found in the [wiki](https://github.com/codehack/go-relax/wiki).

### Use existing or third-party net/http handlers

```go
// split the return of Service.Handler()
path, handler := myservice.Handler()

// now use your handler chain, for example, with a timeout.
http.Handle(path, http.TimeoutHandler(handler, 3, "Time out!"))
```

### Create a new Filter

```go
type MyFilter struct {
	SomeOption string
}
func (f *MyFilter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	// initialize your filter
	if f.SomeOption == "" {
		f.SomeOption = "some value"
	}

	// delegate
	return func(rw relax.ResponseWriter, re *relax.Request) {
		// do stuff before next filter...

		if someError() {
			// error found! stop the request.
			rw.Error(400, "Aborted!")
			return
		}
		// continue to next filter
		next(rw, re)
		// do stuff after previous filters...
	}
}
```

Now place your shinny new filter somewhere in the request chain.

```go
	res.GET("{item}", MyHandler, &MyFilter{})
	res.PUT("{hex:apikey}", KeyHandler, &MyFilter{"Filter APIKey"})
```

## Documentation

The full code documentation is located at GoDoc:

[http://godoc.org/github.com/codehack/go-relax](http://godoc.org/github.com/codehack/go-relax)

**Go-Relax** is Copyright (c) 2014 [Codehack](http://codehack.com).
Published under [MIT License](https://raw.githubusercontent.com/codehack/go-relax/master/LICENSE)



