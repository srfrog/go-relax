// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package override

import (
	"github.com/codehack/go-relax"
	"net/http"
)

// Filter Override changes the Request.Method if the client specifies
// override via HTTP header or query. This allows clients with limited HTTP
// verbs to send REST requests through GET/POST.
type Filter struct {
	// Header expected for HTTP Method override
	// Default: "X-HTTP-Method-Override"
	Header string

	// QueryVar is used if header can't be set
	// Default: "_method"
	QueryVar string

	// Methods specifies the methods can be overridden.
	// Format is Methods["method"] = "override".
	// Default methods:
	//		f.Methods = map[string]string{
	//			"DELETE":  "POST",
	//			"OPTIONS": "GET",
	//			"PATCH":   "POST",
	//			"PUT":     "POST",
	//		}
	Methods map[string]string
}

// Run runs the filter and passes down the following Info:
//
//		ctx.Get("override.method") // method replaced. e.g., "DELETE"
//
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Header == "" {
		f.Header = "X-HTTP-Method-Override"
	}
	if f.QueryVar == "" {
		f.QueryVar = "_method"
	}
	if f.Methods == nil {
		f.Methods = map[string]string{
			"DELETE":  "POST",
			"OPTIONS": "GET",
			"PATCH":   "POST",
			"PUT":     "POST",
		}
	}

	return func(ctx *relax.Context) {
		if override := ctx.Request.URL.Query().Get(f.QueryVar); override != "" {
			ctx.Request.Header.Set(f.Header, override)
		}
		if override := ctx.Request.Header.Get(f.Header); override != "" {
			if override != ctx.Request.Method {
				method, ok := f.Methods[override]
				if !ok {
					ctx.Error(http.StatusBadRequest, override+" method is not overridable.")
					return
				}
				// check that the caller method matches the expected override. e.g., used GET for OPTIONS
				if ctx.Request.Method != method {
					ctx.Error(http.StatusPreconditionFailed, "Must use "+method+" to override "+override)
					return
				}
				ctx.Request.Method = override
				ctx.Request.Header.Del(f.Header)
				ctx.Set("override.method", override)
			}
		}
		next(ctx)
	}
}
