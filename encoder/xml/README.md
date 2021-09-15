# xmlenc

This package provides XML encoding for [Go-Relax](https://github.com/srfrog/go-relax).

## Installation

Using "go get":

	go get "github.com/srfrog/go-relax/encoder/xml"

Then import from source:

	import "github.com/srfrog/go-relax/encoder/xml"

## Usage

To accept and respond with xml, you must add an object to the Service.Encoders map.

```go
package main

import (
	"github.com/srfrog/go-relax"
	"github.com/srfrog/go-relax/encoder/xml"
	"net/http"
)

func main() {
	mysrv := relax.NewService("/api")

	// create and configure new encoder object
	enc := xmlenc.NewEncoder()
	enc.Indented = true

	// assign it to service "mysrv".
	// this maps "application/xml" media queries to this encoder.
	mysrv.Use(enc)

	// done. now you can continue with your resource routes etc...
	mysrv.Run()
}
```

### Options

	encoder := &xmlenc.Encoder{Indented: true, MaxBodySize: 10000, AcceptHeader: "text/xml"}

``Indented``: boolean; set to true to encode indented XML. Default is **false**.

``MaxBodySize``: int; the maximum size (in bytes) of XML content to be read. Default is **4194304** (4MB)

``AcceptHeader``: the MIME media type expected in Accept header. Default is "application/xml"

``ContentTypeHeader``: the MIME media type, and optionally character set, expected in Content-Type header. Default is "application/xml;charset=utf-8"
