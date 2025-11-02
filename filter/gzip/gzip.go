// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package gzip

import (
	"compress/gzip"
	"strings"

	"github.com/srfrog/go-relax"
)

// Filter Gzip compresses the response with gzip encoding, if the client
// indicates support for it.
type Filter struct {
	// CompressionLevel specifies the level of compression used for gzip.
	// Value must be between -1 (gzip.DefaultCompression) to 9 (gzip.BestCompression)
	// A value of 0 (gzip.DisableCompression) will disable compression.
	// Defaults to ``gzip.BestSpeed``
	CompressionLevel int

	// MinLength is the minimum content length, in bytes, required to do compression.
	// Defaults to 100
	MinLength int
}

/*
Run runs the filter and passes down the following Info:

	ctx.Get("content.gzip") // boolean; whether gzip actually happened.

The info passed is used by ETag to generate distinct entity-tags for gzip'ed
content.
*/
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.CompressionLevel == 0 || f.CompressionLevel > gzip.BestCompression {
		f.CompressionLevel = gzip.BestSpeed
	}
	if f.MinLength == 0 {
		f.MinLength = 100
	}
	return func(ctx *relax.Context) {
		// ctx.Set("content.gzip", false)
		ctx.Header().Add("Vary", "Accept-Encoding")

		encodings := ctx.Request.Header.Get("Accept-Encoding")
		if f.CompressionLevel == 0 || !(strings.Contains(encodings, "gzip") || encodings == "*") {
			next(ctx)
			return
		}

		// don't compress ranged responses.
		if ctx.Request.Header.Get("If-Range") != "" {
			next(ctx)
			return
		}

		// Check for encoding preferences.
		if prefs, err := relax.ParsePreferences(encodings); err == nil && len(prefs) > 1 {
			if xgzip, ok := prefs["x-gzip"]; ok {
				prefs["gzip"] = xgzip
			}
			for _, value := range prefs {
				// Client prefers another encoding better, we may support it in another
				// filter. Let that filter handle it instead.
				if value > prefs["gzip"] {
					next(ctx)
					return
				}
			}
		}

		rb := relax.NewResponseBuffer(ctx)
		next(ctx.Clone(rb))
		defer rb.Flush(ctx)

		switch {
		// this might happen when FilterETag runs after GZip
		case rb.Status() == 304:
			ctx.WriteHeader(304)
		case rb.Status() == 204, rb.Status() > 299, rb.Status() < 200:
			break
		case rb.Header().Get("Content-Range") != "":
			break
		case strings.Contains(rb.Header().Get("Content-Encoding"), "gzip"):
			break
		case rb.Len() < f.MinLength:
			break
		default:
			gz, err := gzip.NewWriterLevel(ctx.ResponseWriter, f.CompressionLevel)
			if err != nil {
				return
			}
			defer gz.Close()

			// Only set if gzip actually happened.
			ctx.Set("content.gzip", true)

			rb.Header().Add("Content-Encoding", "gzip")

			// Check if ETag is set, alter it to reflect gzip content.
			if etag := rb.Header().Get("ETag"); etag != "" && !strings.Contains(etag, "gzip") {
				etagGzip := strings.TrimSuffix(etag, `"`) + `-gzip"`
				rb.Header().Set("ETag", etagGzip)
			}

			rb.FlushHeader(ctx.ResponseWriter)
			rb.WriteTo(gz)
			rb.Free()
		}
	}
}
