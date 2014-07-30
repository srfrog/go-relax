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
	contentMediaType     = "application/vnd.relax"
	contentMediaTypeLen  = 21
	contentMediaVersion  = "current"
	contentMediaLanguage = "en_US"
)

// contentFilter implements minimal content negotiation needed for accepting and
// responding to requests. This Filter is required for Service operation.
type contentFilter struct {
	// Encoder must be a pointer in case it's changed at runtime
	enc *map[string]Encoder
}

// Filter info passed down from contentFilter:
//		re.Info.Get("content.encoding") // media type used for encoding
//		re.Info.Get("content.decoding") // Type used in payload requests POST/PUT/PATCH
//		re.Info.Get("content.version") // requested version, or "current"
//		re.Info.Get("content.language") // requested language, or "en_US"
func (self *contentFilter) Run(next HandlerFunc) HandlerFunc {
	return func(rw ResponseWriter, re *Request) {
		// this is our default representation.
		encoder := (*self.enc)["application/json"]
		rw.(*responseWriter).Encode = encoder.Encode
		re.Decode = encoder.Decode

		version, language := contentMediaVersion, contentMediaLanguage

		accept := re.Header.Get("Accept")

		// We check our vendor application MIME for requests of a specific media type.
		// Everything else will default to "application/json" (see above).
		if strings.HasPrefix(accept, contentMediaType) {
			// Accept: application/vnd.relax+{encoding-type}; version=XX; lang=YY
			ct, m, err := mime.ParseMediaType(accept)
			if err != nil {
				rw.Header().Set("Content-Type", encoder.ContentType())
				rw.Error(http.StatusBadRequest, err.Error())
				return
			}
			// check for encoding-type, if client wants a specific format.
			if len(ct) > contentMediaTypeLen {
				if ct[contentMediaTypeLen] != '+' {
					rw.Header().Set("Content-Type", encoder.ContentType())
					rw.Error(http.StatusUnsupportedMediaType, "That media type is not supported.")
					return
				}
				mime := mime.TypeByExtension("." + ct[contentMediaTypeLen+1:])
				if _, ok := (*self.enc)[mime]; !ok {
					rw.Header().Set("Content-Type", encoder.ContentType())
					rw.Error(http.StatusNotAcceptable, "That content type is not supported for response.")
					return
				}
				encoder = (*self.enc)[mime]
				rw.(*responseWriter).Encode = encoder.Encode
			}

			if v, ok := m["version"]; ok {
				version = v
			}
			if v, ok := m["lang"]; ok {
				language = v
			}
		}

		// BUG(TODO): contentFilter add support for URI file-extension format ?

		// at this point we know which encoding format the client expects
		rw.Header().Set("Content-Type", encoder.ContentType())
		re.Info.Set("content.encoding", encoder.Accept())
		re.Info.Set("content.version", version)
		re.Info.Set("content.language", language)

		// We now check for Content-Type of payload if the method POST, PUT or PATCH.
		if re.Method[0] == 'P' {
			// Content-Type: application/{encoding-type}
			ct, _, err := mime.ParseMediaType(re.Header.Get("Content-Type"))
			if err != nil {
				rw.Error(http.StatusBadRequest, err.Error())
				return
			}
			decoder := (*self.enc)[ct]
			if decoder == nil {
				rw.Error(http.StatusNotAcceptable, "That content type is not supported for transfer.")
				return
			}
			re.Decode = decoder.Decode
			re.Info.Set("content.decoding", ct)
		}

		next(rw, re)
	}
}
