// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"mime"
	"net/http"
	"strings"
)

const (
	contentMimeType        = "application/vnd.relax"
	contentMimeTypeLen     = 21
	contentDefaultVersion  = "current"
	contentDefaultLanguage = "en_US"
)

// contentFilter implements minimal content negotiation needed for accepting and
// responding to requests. This Filter is required for Service operation.
type contentFilter struct {
	// Encoder must be a pointer in case it's changed at runtime
	enc *Encoder
}

// Filter info passed down from contentFilter:
//		re.Info.Get("content.encoding") // MIME type used for encoding
//		re.Info.Get("content.version") // requested version, or "current"
//		re.Info.Get("content.language") // requested language, or "en_US"
func (self *contentFilter) Run(next HandlerFunc) HandlerFunc {
	return func(rw ResponseWriter, re *Request) {
		re.Info.Set("content.encoding", (*self.enc).Accept())
		re.Info.Set("content.version", contentDefaultVersion)
		re.Info.Set("content.language", contentDefaultLanguage)

		accept := re.Header.Get("Accept")

		// Accept: application/vnd.relax+{encoding-type}; version=XX; lang=YY
		if strings.HasPrefix(accept, contentMimeType) {
			ct, m, err := mime.ParseMediaType(accept)
			if err != nil {
				rw.Error(http.StatusBadRequest, err.Error())
				return
			}
			// check for encoding-type
			if len(ct) > contentMimeTypeLen {
				if ct[contentMimeTypeLen] != '+' {
					rw.Error(http.StatusUnsupportedMediaType, "That media type is not supported.")
					return
				}
				if mime.TypeByExtension("."+ct[contentMimeTypeLen+1:]) != (*self.enc).Accept() {
					rw.Error(http.StatusNotAcceptable, "That encoding is not supported.")
					return
				}
			}
			if v, ok := m["version"]; ok {
				re.Info.Set("content.version", v)
			}
			if v, ok := m["lang"]; ok {
				re.Info.Set("content.language", v)
			}
		}

		// Content-Type: application/{encoding-type}
		if re.Method[0] == 'P' { // POST, PUT, PATCH
			ct, _, err := mime.ParseMediaType(re.Header.Get("Content-Type"))
			if err != nil {
				rw.Error(http.StatusBadRequest, err.Error())
				return
			}
			if ct != (*self.enc).Accept() {
				rw.Error(http.StatusNotAcceptable, "That encoding is not supported.")
				return
			}
		}

		next(rw, re)
	}
}
