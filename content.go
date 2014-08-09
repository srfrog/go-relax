// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"mime"
	"net/http"
	"strconv"
	"strings"
)

const (
	// ContentMediaType is the vendor extended media type used by this framework.
	ContentMediaType    = "application/vnd.relax"
	contentMediaTypeLen = 21

	// ContentDefaultVersion is the default version value when no content version is requested.
	ContentDefaultVersion = "current"

	// ContentDefaultLanguage is the default langauge value when no content language is requested.
	ContentDefaultLanguage = "en-US"
)

/*
Content does content negotiation to select the supported representations
for the request and response. The default representation uses media type
"application/json" which is our default media encoding. If new media types are
available to the service, a client can request it using the Accept header.
The format of the Accept header uses the following vendor extension:

	Accept: application/vnd.relax+{subtype}; version=XX; lang=YY

The values for {subtype}, {version} and {lang} are optional. They correspond
in order; to media subtype, content version and, language. If any value is missing
or unsupported the default values are used. If a request Accept header is not
using the vendor extension, the default values are used:

	Accept: application/vnd.relax+json; version="current"; lang="en"

When Accept indicates all media types, the media subtype can be requested
through the URL path's extension. If the service doesn't support the media encoding,
then it will respond with an HTTP error code.

	GET /api/v1/tickets.xml
	GET /company/users/123.json

Note that the extension should be appended to a collection or a resource item.
The extension is removed before the request is dispatched to the routing engine.

If the request header Accept-Language is found, the value for content language
is automatically set to that. The underlying application should use this to
construct a proper respresentation in that language.

Content passes down the following info to filters:

	ctx.Info.Get("content.encoding") // media type used for encoding
	ctx.Info.Get("content.decoding") // Type used in payload requests POST/PUT/PATCH
	ctx.Info.Get("content.version")  // requested version, or "current"
	ctx.Info.Get("content.language") // requested language, or "en_US"

Requests and responses can use mixed representations if the service supports the
media types.

See also, http://tools.ietf.org/html/rfc5646; tags to identify languages.
*/
func (svc *Service) Content(next HandlerFunc) HandlerFunc {
	var alt struct {
		Alternatives []string `json:"alternatives"`
	}

	return func(ctx *Context) {
		// This is our default representation.
		encoder := svc.encoders["application/json"]
		ctx.Encode = encoder.Encode
		ctx.Decode = encoder.Decode

		version, language := ContentDefaultVersion, ContentDefaultLanguage

		accept := ctx.Request.Header.Get("Accept")

		if accept == "*/*" {
			// Check if subtype is in the requested URL path's extension.
			// Path: /api/v1/users.xml
			if ext := PathExt(ctx.Request.URL.Path); ext != "" {
				// remove extension from path.
				ctx.Request.URL.Path = strings.TrimSuffix(ctx.Request.URL.Path, ext)
				// create vendor media type and fallthrough
				accept = ContentMediaType + "+" + ext[1:]
			}
		}

		// We check our vendor media type for requests of a specific subtype.
		// Everything else will default to "application/json" (see above).
		if strings.HasPrefix(accept, ContentMediaType) {
			// Accept: application/vnd.relax+{subtype}; version={version}; lang={lang}
			ct, m, err := mime.ParseMediaType(accept)
			if err != nil {
				ctx.Header().Set("Content-Type", encoder.ContentType())
				ctx.Error(http.StatusBadRequest, err.Error())
				return
			}
			// check for encoding-type, if client wants a specific format.
			if len(ct) > contentMediaTypeLen && ct[contentMediaTypeLen] == '+' {
				tbe := mime.TypeByExtension("." + ct[contentMediaTypeLen+1:])
				if svc.encoders[tbe] == nil {
					alt.Alternatives = nil
					for _, enc := range svc.encoders {
						alt.Alternatives = append(alt.Alternatives, enc.Accept())
					}
					ctx.Header().Set("Content-Type", encoder.ContentType())
					ctx.Error(http.StatusNotAcceptable, "That media type is not supported for response.", alt)
					return
				}
				encoder = svc.encoders[tbe]
				ctx.Encode = encoder.Encode
			}

			if v, ok := m["version"]; ok {
				version = v
			}
			if v, ok := m["lang"]; ok {
				language = v
			}
		}

		// Check for language preferences.
		if langrange := ctx.Request.Header.Get("Accept-Language"); langrange != "" {
			// Accept-Language: da, jp;q=0.8, en;q=0.9
			prefs, err := ParsePreferences(langrange)
			// If language parsing fails, continue with request. But we still log it.
			// See https://tools.ietf.org/html/rfc7231#section-5.3.5
			if err != nil {
				Log.Println(LogDebug, "Language parsing failed:", err.Error())
			} else {
				// If content language is not listed, give it a competitive value for sanity.
				// The value most likely is still "en" (English).
				if _, ok := prefs[language]; !ok {
					prefs[language] = 0.85
				}
				// Notice that we completely ignore language priority, since Go maps list randomly.
				for code, value := range prefs {
					if value > prefs[language] {
						language = code
					}
				}
			}
		}

		// At this point we know the response media type.
		ctx.Header().Set("Content-Type", encoder.ContentType())
		ctx.Info.Set("content.encoding", encoder.Accept())
		ctx.Info.Set("content.version", version)
		ctx.Info.Set("content.language", language)

		// Now check for payload representation for unsafe methods: POST PUT PATCH.
		if ctx.Request.Method[0] == 'P' {
			// Content-Type: application/{subtype}
			ct, _, err := mime.ParseMediaType(ctx.Request.Header.Get("Content-Type"))
			if err != nil {
				ctx.Error(http.StatusBadRequest, err.Error())
				return
			}
			decoder := svc.encoders[ct]
			if decoder == nil {
				ctx.Error(http.StatusUnsupportedMediaType, "That media type is not supported for transfer.")
				return
			}
			ctx.Decode = decoder.Decode
			ctx.Info.Set("content.decoding", ct)
		}

		next(ctx)
	}
}

/*
PathExt returns the media subtype extension in an URL path.
The extension begins from the last dot:

	/api/v1/tickets.xml => ".xml"

Returns the extension with dot, or empty string "" if not found.
*/
func PathExt(path string) string {
	dot := strings.LastIndex(path, ".")
	if dot > -1 {
		return path[dot:]
	}
	return ""
}

// ParsePreferences is a very naive and simple parser for header value preferences.
// Returns a map of preference=quality values for each preference with a quality value.
// If a preference doesn't specify quality, then a value of 1.0 is assumed (bad!).
// If the quality float value can't be parsed from string, an error is returned.
func ParsePreferences(values string) (map[string]float32, error) {
	prefs := make(map[string]float32, 0)
	for _, rawval := range strings.Split(values, ",") {
		val := strings.SplitN(strings.TrimSpace(rawval), ";q=", 2)
		prefs[val[0]] = 1.0
		if len(val) == 2 {
			f, err := strconv.ParseFloat(val[1], 32)
			if err != nil {
				return nil, err
			}
			prefs[val[0]] = float32(f)
		}
	}
	return prefs, nil
}

func init() {
	// just in case
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".xml", "application/xml")
}
