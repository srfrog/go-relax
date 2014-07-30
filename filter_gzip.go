// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"compress/gzip"
	"strings"
)

// FilterGzip compresses the response with gzip encoding, if the client
// supports it.
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

// Run runs the filter and passes down the following Info:
//		re.Info.Get("compress.type") // compression method used. e.g., "gzip"
func (self *FilterGzip) Run(next HandlerFunc) HandlerFunc {
	if self.CompressionLevel == 0 || self.CompressionLevel > gzip.BestCompression {
		self.CompressionLevel = gzip.BestSpeed
	}
	if self.MinLength == 0 {
		self.MinLength = 100
	}
	return func(rw ResponseWriter, re *Request) {
		rw.Header().Add("Vary", "Accept-Encoding")

		// BUG(TODO): FilterGzip is not checking header values for qvalue or identity
		h := re.Header.Get("Accept-Encoding")
		if self.CompressionLevel == 0 || !(strings.Contains(h, "gzip") || h == "*") {
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (h=%q)", re.Info.Get("context.request_id"), h)
			next(rw, re)
			return
		}

		// already gzipped
		if strings.Contains(re.Header.Get("Content-Encoding"), "gzip") || re.Header.Get("Content-Range") != "" {
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (unsupported)", re.Info.Get("context.request_id"))
			next(rw, re)
			return
		}

		rb, buf := NewResponseBuffer(rw)
		next(rb, re)

		if buf.Status() == 204 || buf.Status() < 200 || buf.Status() > 299 {
			Log.Printf(LOG_DEBUG, "%s FilterGzip: compression disabled (status=%d)", re.Info.Get("context.request_id"), buf.Status())
			goto Finish
		}

		if n := buf.Len(); n > self.MinLength && n < 0xffff {
			gz, err := gzip.NewWriterLevel(rw, self.CompressionLevel)
			defer gz.Close()
			if err != nil {
				Log.Println(LOG_CRIT, "FilterGzip: compression failed:", err.Error())
			} else {
				re.Info.Set("compress.type", "gzip")
				buf.Header().Add("Content-Encoding", "gzip")
				buf.WriteTo(gz)
				return
			}
		}
	Finish:
		buf.Flush()
	}
}
