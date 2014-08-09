// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"compress/gzip"
	"strings"
)

// FilterGzip compresses the response with gzip encoding, if the client
// indicates support for it.
type FilterGzip struct {
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

	ctx.Info.Get("content.gzip") // boolean; whether gzip actually happened.

The info passed is used by ETag to generate distinct entity-tags for gzip'ed
content.
*/
func (f *FilterGzip) Run(next HandlerFunc) HandlerFunc {
	if f.CompressionLevel == 0 || f.CompressionLevel > gzip.BestCompression {
		f.CompressionLevel = gzip.BestSpeed
	}
	if f.MinLength == 0 {
		f.MinLength = 100
	}
	return func(ctx *Context) {
		ctx.Info.Set("content.gzip", false)
		ctx.Header().Add("Vary", "Accept-Encoding")

		// BUG(TODO): FilterGzip is not checking header values for qvalue or identity
		h := ctx.Request.Header.Get("Accept-Encoding")
		if f.CompressionLevel == 0 || !(strings.Contains(h, "gzip") || h == "*") {
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (h=%q)", ctx.Info.Get("context.request_id"), h)
			next(ctx)
			return
		}

		// already gzipped
		if ctx.Request.Header.Get("If-Range") != "" {
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (range)", ctx.Info.Get("context.request_id"))
			next(ctx)
			return
		}

		next(ctx.Capture()) // start buffering

		switch {
		// this might happen when FilterETag runs after GZip
		case ctx.Buffer.Status() == 304:
			ctx.WriteHeader(304)
			break
		case ctx.Buffer.Status() == 204, ctx.Buffer.Status() > 299, ctx.Buffer.Status() < 200:
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (status=%d)", ctx.Info.Get("context.request_id"), ctx.Buffer.Status())
			break
		case ctx.Buffer.Header().Get("Content-Range") != "":
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (content-range)", ctx.Info.Get("context.request_id"))
			break
		case strings.Contains(ctx.Buffer.Header().Get("Content-Encoding"), "gzip"):
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (already gzip'ed)", ctx.Info.Get("context.request_id"))
			break
		case ctx.Buffer.Len() < f.MinLength:
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (content too small)", ctx.Info.Get("context.request_id"))
			break
		default:
			gz, err := gzip.NewWriterLevel(ctx.ResponseWriter, f.CompressionLevel)
			if err != nil {
				Log.Println(LOG_CRIT, "FilterGzip: compression failed:", err.Error())
				break
			} else {
				defer gz.Close()
				ctx.Buffer.Header().Add("Content-Encoding", "gzip")

				// Only set if gzip actually happened.
				ctx.Info.Set("content.gzip", true)

				// Check if ETag is set, alter it to reflect gzip content.
				if etag := ctx.Buffer.Header().Get("ETag"); etag != "" && !strings.Contains(etag, "gzip") {
					etagGzip := strings.TrimSuffix(etag, `"`) + `-gzip"`
					ctx.Buffer.Header().Set("ETag", etagGzip)
				}

				ctx.Buffer.FlushHeader(ctx.ResponseWriter)
				ctx.Buffer.WriteTo(gz)
				ctx.Buffer.Free()
				ctx.Buffer = nil // stop Context.Relase from flushing
			}
		}

		ctx.Release() // finish buffering
	}
}
