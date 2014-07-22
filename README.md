# Go-Relax [![GoDoc](https://godoc.org/github.com/codehack/go-relax?status.svg)](https://godoc.org/github.com/codehack/go-relax) ![Project progress](http://progressed.io/bar/32 "Progress")

*Build fast and efficient RESTful APIs in [Go](http://golang.org)*

**Go-Relax** is a framework of pluggable components to build RESTful API's. It provides a thin layer over ``net/http`` to serve resources, without imposing a rigid structure. It is meant to be used along ``http.ServeMux``, but will work as a replacement as it implements ``http.Handler``.

*Go-Relax* wraps itself around the concept of resources. A resource is any object that can serve requests (data) to clients. *Go-Relax* tries to be more than a router to resources, but rather a _resource service_.

## Features

- Helps build API's that follow the REST concept using ROA architecture.
- It follows REST best practices, with inspiration from other REST API's like Heroku and GitHub's.
- Works fine along with ``http.ServeMux`` or independently as ``http.Handler``.
- Uses JSON encoding by default, enforcing content-negotiation per request.
- Default routing engine uses **trie with regexp matching** for speed and flexibility.
- Includes filters used by most API's. aka "Batteries included" (WIP)
- All framework components: encoding, routing, logging and filters, are modular. Easily replaced by external packages.
- Uses ``sync.pool`` to efficiently use resources when under heavy load.

## Installation

Using "go get":

	go get github.com/codehack/go-relax

Then import from source:

	import "github.com/codehack/go-relax"

## Example

Check [example_test.go](https://github.com/codehack/go-relax/blob/master/example_test.go) for an example of basic usage.

## Howto

### Use existing or third-party net/http handlers

```go
// split the return of Handler()
path, handler := res.Handler()

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



