// Copyright 2016 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package multipart

import (
	"mime"
	"net/http"
	"path/filepath"

	"github.com/codehack/go-relax"
)

// DefaultMaxMemory is 4 MiB for storing a request body in memory.
const (
	DefaultMaxMemory = 1 << 22
)

// Filter Multipart handles multipart file uploads via a specific path.
type Filter struct {
	// MaxMemory total bytes of the request body that is stored in memory.
	// Increase this value if you expect large documents.
	// Default: 4 MiB
	MaxMemory int64
}

// Run runs the filter and passes down the following Info:
//
//		ctx.Get("multipart.files") // list of files processed (*[]*FileHeader)
//
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.MaxMemory == 0 {
		f.MaxMemory = DefaultMaxMemory
	}

	return func(ctx *relax.Context) {
		if ctx.Request.Method != "POST" {
			next(ctx)
			return
		}

		ct, _, err := mime.ParseMediaType(ctx.Request.Header.Get("Content-Type"))
		if err != nil {
			ctx.Error(http.StatusBadRequest, err.Error())
			return
		}

		if ct != "multipart/form-data" {
			ctx.Error(http.StatusUnsupportedMediaType,
				"That media type is not supported for transfer.",
				"Expecting multipart/form-data")
			return
		}

		if err := ctx.Request.ParseMultipartForm(f.MaxMemory); err != nil {
			ctx.Error(http.StatusBadRequest, err.Error())
			return
		}

		files, ok := ctx.Request.MultipartForm.File["files"]
		if !ok {
			ctx.Error(http.StatusBadRequest, "insufficient parameters")
			return
		}

		for i := range files {
			ext := filepath.Ext(filepath.Base(filepath.Clean(files[i].Filename)))
			if ext == "" {
				ctx.Error(http.StatusBadRequest, "could not get the file extension")
				return
			}

			if mime.TypeByExtension(ext) == "" {
				ctx.Error(http.StatusBadRequest, "file type is unknown")
				return
			}
		}

		ctx.Set("multipart.files", &files)

		next(ctx)
	}
}

// RunIn implements the LimitedFilter interface. This will limit this filter
// to run only for router paths, not resources or service.
func (f *Filter) RunIn(e interface{}) bool {
	switch e.(type) {
	case relax.Router:
		return true
	}
	return false
}
