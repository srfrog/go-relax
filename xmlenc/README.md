# xmlenc

This package provides XML encoding for [Go-Relax](https://github.com/codehack/go-relax).

## Installation

Using "go get":

	go get "github.com/codehack/go-relax/xmlenc"

Then import from source:

	import "github.com/codehack/go-relax/xmlenc"

## Usage

For most programs you'll change the service default encoder (JSON) to this one.

```go
package main

import (
	"github.com/codehack/go-relax"
	"github.com/codehack/go-relax/xmlenc"
	"net/http"
)

func main() {
	mysrv := relax.NewService("/api")

	mysrv.Encoding(&xmlenc.EncoderXML{Indented: true})

	// ... your resource routes etc...

	http.Handle(mysrv.Handler())
	log.Fatal(http.ListenAndServe(":8000", nil))
}
```

### Options

	encoder := &xmlenc.EncoderXML{Indented: true, MaxBodySize: 10000}

``Indented``: boolean; set to true to encode indented XML. Default is **false**.

``MaxBodySize``: int; the maximum size (in bytes) of XML content to be read. Default is **2097152** (2MB)
