# Go-Relax [![GoDoc](https://godoc.org/github.com/codehack/go-relax?status.svg)](https://godoc.org/github.com/codehack/go-relax)

*Build fast and efficient RESTful APIs in [Go](http://golang.org)*

**Go-Relax** is a framework of pluggable components to build RESTful API's. It provides a thin layer over net/http to serve resources, without imposing a rigid structure. It is meant to be used along http.ServeMux, but will work as a replacement as it implements http.Handler.

## Features

- Helps build API's that follow the REST concept using ROA architecture.
- It follows REST best practices, with inspiration from other REST API's like Heroku and GitHub's.
- Works fine along with http.ServeMux or independently as http.Handler.
- Uses JSON encoding by default, enforcing content-negotiation per request.
- Default routing engine uses trie with regexp matching for speed and flexibility.
- Includes filters used by most API's. aka "Batteries included"
- All framework components: encoding, routing, logging and filters, are modular. Easily replaced by external packages.
- Uses ``sync.pool`` to efficiently use resources when under heavy load.

## Installation

Using "go get":

	go get github.com/codehack/go-relax

Then import from source:

	import "github.com/codehack/go-relax"

## Documentation

The full code documentation is located at GoDoc:

[http://godoc.org/github.com/codehack/go-relax](http://godoc.org/github.com/codehack/go-relax)

**Go-Environ** is Copyright (c) 2014 [Codehack](http://codehack.com).
Published under [MIT License](https://raw.githubusercontent.com/codehack/go-relax/master/LICENSE)



