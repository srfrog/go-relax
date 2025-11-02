// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package etag

import (
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/srfrog/go-relax"
)

// Filter ETag generates an entity-tag header "ETag" for body content of a response.
// It will use pre-generated etags from the underlying filters or handlers, if available.
// Optionally, it will also handle the conditional response based on If-Match
// and If-None-Match checks on specific entity-tag values.
// This implementation follows the recommendation in http://tools.ietf.org/html/rfc7232
type Filter struct {
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
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	return func(ctx *relax.Context) {
		var etag string

		// Start a buffered context. All writes are diverted to a ResponseBuffer.
		rb := relax.NewResponseBuffer(ctx)
		next(ctx.Clone(rb))
		defer rb.Flush(ctx)

		// Do not pass GO. Do not collect $200
		if rb.Status() < 200 || rb.Status() == http.StatusNoContent ||
			(rb.Status() > 299 && rb.Status() != http.StatusPreconditionFailed) ||
			!strings.Contains("DELETE GET HEAD PATCH POST PUT", ctx.Request.Method) {
			goto Finish
		}

		etag = rb.Header().Get("ETag")

		if isEtagMethod(ctx.Request.Method) && rb.Status() == http.StatusOK {
			if etag == "" {
				alter := ""
				// Change etag when using content encoding.
				if ce := rb.Header().Get("Content-Encoding"); ce != "" {
					alter = "-" + ce
				}
				h := sha1.New()
				h.Write(rb.Bytes())
				etag = `"` + hex.EncodeToString(h.Sum(nil)) + alter + `"`
			}
		}

		if !f.DisableConditionals {
			// If-Match
			ifmatch := ctx.Request.Header.Get("If-Match")
			if ifmatch != "" && ((ifmatch == "*" && etag == "") || !etagStrongCmp(ifmatch, etag)) {
				/*
					// FIXME: need to verify Status per request.
					if strings.Contains("DELETE PATCH POST PUT", ctx.Request.Method) && rb.Status() != http.StatusPreconditionFailed {
						// XXX: we cant confirm it's the same resource item without re-GET'ing it.
						// XXX: maybe etag should be changed from strong to weak.
						etag = ""
						goto Finish
					}
				*/
				ctx.WriteHeader(http.StatusPreconditionFailed)
				rb.Free()
				return
			}

			// If-Unmodified-Since
			ifunmod := ctx.Request.Header.Get("If-Unmodified-Since")
			if ifmatch == "" && ifunmod != "" {
				modtime, _ := time.Parse(http.TimeFormat, ifunmod)
				lastmod, _ := time.Parse(http.TimeFormat, rb.Header().Get("Last-Modified"))
				if !modtime.IsZero() && !lastmod.IsZero() && lastmod.After(modtime) {
					ctx.WriteHeader(http.StatusPreconditionFailed)
					rb.Free()
					return
				}
			}

			// If-None-Match
			ifnone := ctx.Request.Header.Get("If-None-Match")
			if ifnone != "" && ((ifnone == "*" && etag != "") || etagWeakCmp(ifnone, etag)) {
				// defer rb.Reset()
				if isEtagMethod(ctx.Request.Method) {
					rb.Header().Set("ETag", etag)
					rb.Header().Add("Vary", "If-None-Match")
					rb.WriteHeader(http.StatusNotModified)
					rb.Reset()
					return
				}
				ctx.WriteHeader(http.StatusPreconditionFailed)
				rb.Free()
				return
			}

			// If-Modified-Since
			ifmods := ctx.Request.Header.Get("If-Modified-Since")
			if ifnone == "" && ifmods != "" && !isEtagMethod(ctx.Request.Method) {
				modtime, _ := time.Parse(http.TimeFormat, ifmods)
				lastmod, _ := time.Parse(http.TimeFormat, rb.Header().Get("Last-Modified"))
				if !modtime.IsZero() && !lastmod.IsZero() && (lastmod.Before(modtime) || lastmod.Equal(modtime)) {
					if etag != "" {
						rb.Header().Set("ETag", etag)
						rb.Header().Add("Vary", "If-None-Match")
					}
					rb.Header().Add("Vary", "If-Modified-Since")
					rb.WriteHeader(http.StatusNotModified)
					rb.Reset()
					return
				}
			}
		}

	Finish:
		if etag != "" {
			rb.Header().Set("ETag", etag)
			rb.Header().Add("Vary", "If-None-Match")
		}
	}
}

func isEtagMethod(m string) bool {
	return m == "GET" || m == "HEAD"
}
