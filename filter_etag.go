// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// FilterETag generates an entity-tag header "ETag" for body content of a response.
// It will use pre-generated etags from the underlying filters or handlers, if availble.
// Optionally, it will also handle the conditional response based on If-Match
// and If-None-Match checks on specific entity-tag values.
// This implementation follows the recommendation in http://tools.ietf.org/html/rfc7232
type FilterETag struct {
	// DisableConditionals will make this filter ignore the values from the headers
	// If-None-Match and If-Match and not do conditional entity tests. An ETag will
	// still be generated, if possible.
	// Defaults to false
	DisableConditionals bool
}

// etagStrongCmp does strong comparison of If-Match entity values.
func etagStrongCmp(etags, etag string) bool {
	if etag == "" || strings.HasPrefix(etag, "W/") {
		return false
	}
	for _, v := range strings.Split(etags, ",") {
		if strings.TrimSpace(v) == etag {
			return true
		}
	}
	return false
}

// etagWeakCmp does weak comparison of If-None-Match entity values.
func etagWeakCmp(etags, etag string) bool {
	if etag == "" {
		return false
	}
	return strings.Contains(etags, strings.Trim(etag, `"`))
}

// Run runs the filter and passes down the following Info:
//		ctx.Info.Get("etag.enabled") // boolean; true if etag is enabled (always)
func (f *FilterETag) Run(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		var etag string
		ctx.Info.Set("etag.enabled", true)

		next(ctx.Capture())
		defer ctx.Release()

		// Do not pass GO. Do not collect $200
		if ctx.Buffer.Status() < 200 || ctx.Buffer.Status() == http.StatusNoContent ||
			(ctx.Buffer.Status() > 299 && ctx.Buffer.Status() != http.StatusPreconditionFailed) ||
			!strings.Contains("DELETE GET HEAD PATCH POST PUT", ctx.Request.Method) {
			Log.Printf(LogDebug, "%s FilterETag: no ETag generated (status=%d method=%s)", ctx.Info.Get("context.request_id"), ctx.Buffer.Status(), ctx.Request.Method)
			goto Finish
		}

		etag = ctx.Buffer.Header().Get("ETag")

		if (ctx.Request.Method == "GET" || ctx.Request.Method == "HEAD") && ctx.Buffer.Status() == http.StatusOK {
			if etag == "" {
				alter := ""
				// Change etag when using content encoding.
				// XXX: support multiple encodings?
				if ce := ctx.Buffer.Header().Get("Content-Encoding"); ce != "" {
					alter = "-" + ce
				}
				etag = fmt.Sprintf(`"%x%s"`, sha1.Sum(ctx.Buffer.Bytes()), alter)
			}
		}

		if !f.DisableConditionals {
			// If-Match
			ifmatch := ctx.Request.Header.Get("If-Match")
			if ifmatch != "" && ((ifmatch == "*" && etag == "") || !etagStrongCmp(ifmatch, etag)) {
				/*
					// FIXME: need to verify Status per request.
					if strings.Contains("DELETE PATCH POST PUT", ctx.Request.Method) && ctx.Buffer.Status() != http.StatusPreconditionFailed {
						// XXX: we cant confirm it's the same resource item without re-GET'ing it.
						// XXX: maybe etag should be changed from strong to weak.
						etag = ""
						Log.Printf(LogDebug, "%s FilterETag: no ETag generated for match (status=%d method=%s)", ctx.Info.Get("context.request_id"), ctx.Buffer.Status(), ctx.Request.Method)
						goto Finish
					}
				*/
				ctx.WriteHeader(http.StatusPreconditionFailed)
				ctx.Buffer.Free()
				return
			}

			// If-Unmodified-Since
			ifunmod := ctx.Request.Header.Get("If-Unmodified-Since")
			if ifmatch == "" && ifunmod != "" {
				modtime, _ := time.Parse(time.RFC1123, ifunmod)
				lastmod, _ := time.Parse(time.RFC1123, ctx.Buffer.Header().Get("Last-Modified"))
				if !modtime.IsZero() && !lastmod.IsZero() && lastmod.After(modtime) {
					ctx.WriteHeader(http.StatusPreconditionFailed)
					ctx.Buffer.Free()
					return
				}
			}

			// If-None-Match
			ifnone := ctx.Request.Header.Get("If-None-Match")
			if ifnone != "" && ((ifnone == "*" && etag != "") || etagWeakCmp(ifnone, etag)) {
				// defer ctx.Buffer.Reset()
				if ctx.Request.Method == "GET" || ctx.Request.Method == "HEAD" {
					ctx.Buffer.Header().Set("ETag", etag)
					ctx.Buffer.Header().Add("Vary", "If-None-Match")
					ctx.Buffer.WriteHeader(http.StatusNotModified)
					ctx.Buffer.Reset()
					return
				}
				ctx.WriteHeader(http.StatusPreconditionFailed)
				ctx.Buffer.Free()
				return
			}

			// If-Modified-Since
			ifmods := ctx.Request.Header.Get("If-Modified-Since")
			if ifnone == "" && ifmods != "" && !(ctx.Request.Method == "GET" || ctx.Request.Method == "HEAD") {
				modtime, _ := time.Parse(time.RFC1123, ifmods)
				lastmod, _ := time.Parse(time.RFC1123, ctx.Buffer.Header().Get("Last-Modified"))
				if !modtime.IsZero() && !lastmod.IsZero() && (lastmod.Before(modtime) || lastmod.Equal(modtime)) {
					if etag != "" {
						ctx.Buffer.Header().Set("ETag", etag)
						ctx.Buffer.Header().Add("Vary", "If-None-Match")
					}
					ctx.Buffer.Header().Add("Vary", "If-Modified-Since")
					ctx.Buffer.WriteHeader(http.StatusNotModified)
					ctx.Buffer.Reset()
					return
				}
			}
		}

	Finish:
		if etag != "" {
			ctx.Buffer.Header().Set("ETag", etag)
			ctx.Buffer.Header().Add("Vary", "If-None-Match")
		}
	}
}
