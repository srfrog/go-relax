// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"mime"
	"net/http"
	"strings"
)

const (
	defaultMediatype = "application/vnd.codehack.relax"

	defaultVersion = "current"

	defaultLanguage = "en-US"
)

/*
Content does content negotiation to select the supported representations
for the request and response. The default representation uses media type
"application/json". If new media types are available to the service, a client
can request it via the Accept header. The format of the Accept header uses
the following vendor extension:

	Accept: application/vnd.relax+{subtype}; version={version}; lang={language}

The values for {subtype}, {version} and {language} are optional. They correspond
in order; to media subtype, content version and, language. If any value is missing
or unsupported the default values are used. If a request Accept header is not
using the vendor extension, the default values are used:

	Accept: application/vnd.relax+json; version="current"; lang="en"

By decoupling version and lang from the media type, it allows us to have separate
versions for the same resource and with individual language coverage.

When Accept indicates all media types "*&#5C;*", the media subtype can be requested
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

	ctx.Get("content.encoding") // media type used for encoding
	ctx.Get("content.decoding") // Type used in payload requests POST/PUT/PATCH
	ctx.Get("content.version")  // requested version, or "current"
	ctx.Get("content.language") // requested language, or "en-US"

Requests and responses can use mixed representations if the service supports the
media types.

See also, http://tools.ietf.org/html/rfc5646; tags to identify languages.
*/
var Content struct {
	// MediaType is the vendor extended media type used by this framework.
	// Default: application/vnd.codehack.relax
	Mediatype string
	// Version is the version used when no content version is requested.
	// Default: current
	Version string
	// Language is the language used when no content language is requested.
	// Default: en-US
	Language string
}

// content is the function that does the actual content-negotiation described above.
func (svc *Service) content(next HandlerFunc) HandlerFunc {
	// JSON is our default representation.
	json := svc.encoders["application/json"]

	return func(ctx *Context) {
		ctx.Encode = json.Encode
		ctx.Decode = json.Decode

		encoder := json

		version := acceptVersion(ctx.Request.Header.Get("Accept-Version"))

		language := acceptLanguage(ctx.Request.Header.Get("Accept-Language"))

		accept := ctx.Request.Header.Get("Accept")
		if accept == "*/*" {
			// Check if subtype is in the requested URL path's extension.
			// Path: /api/v1/users.xml
			if ext := PathExt(ctx.Request.URL.Path); ext != "" {
				// remove extension from path.
				ctx.Request.URL.Path = strings.TrimSuffix(ctx.Request.URL.Path, ext)
				// create vendor media type and fallthrough
				accept = Content.Mediatype + "+" + ext[1:]
			}
		}

		// We check our vendor media type for requests of a specific subtype.
		// Everything else will default to "application/json" (see above).
		if strings.HasPrefix(accept, Content.Mediatype) {
			// Accept: application/vnd.relax+{subtype}; version={version}; lang={lang}
			mt, op, err := mime.ParseMediaType(accept)
			if err != nil {
				ctx.Header().Set("Content-Type", json.ContentType())
				ctx.Error(http.StatusBadRequest, err.Error())
				return
			}
			// check for media subtype (encoding) request.
			if idx := strings.Index(mt, "+"); idx != -1 {
				tbe := mime.TypeByExtension("." + mt[idx+1:])
				enc, ok := svc.encoders[tbe]
				if !ok {
					ctx.Header().Set("Content-Type", json.ContentType())
					ctx.Error(http.StatusNotAcceptable,
						"That media type is not supported for response.",
						"You may use type '"+json.Accept()+"'")
					return
				}
				encoder = enc
				ctx.Encode = encoder.Encode
			}

			// If version or language were specified they are preferred over Accept-* headers.
			if v, ok := op["version"]; ok {
				version = v
			}
			if v, ok := op["lang"]; ok {
				language = v
			}
		}

		// At this point we know the response media type.
		ctx.Header().Set("Content-Type", encoder.ContentType())

		// Pass the info down to other handlers.
		ctx.Set("content.encoding", encoder.Accept())
		ctx.Set("content.version", version)
		ctx.Set("content.language", language)

		// Now check for payload representation for unsafe methods: POST PUT PATCH.
		if ctx.Request.Method[0] == 'P' {
			// Content-Type: application/{subtype}
			ct, _, err := mime.ParseMediaType(ctx.Request.Header.Get("Content-Type"))
			if err != nil {
				ctx.Error(http.StatusBadRequest, err.Error())
				return
			}
			decoder, ok := svc.encoders[ct]
			if !ok {
				ctx.Error(http.StatusUnsupportedMediaType,
					"That media type is not supported for transfer.",
					"You may use type '"+json.Accept()+"'")
				return
			}
			ctx.Decode = decoder.Decode
			ctx.Set("content.decoding", ct)
		}

		next(ctx)
	}
}

// acceptVersion checks for specific version in Accept-Version HTTP header.
// returns the version requested or Content.Version if none is set.
//
// Accept-Version: v1
func acceptVersion(version string) string {
	if version == "" {
		return Content.Version
	}
	return version
}

// acceptLanguage checks for language preferences in Accept-Language header.
// It returns the language code with highest quality. If none are set, returns
// Content.Language global default.
//
// Accept-Language: da, jp;q=0.8, en;q=0.9
func acceptLanguage(value string) string {
	if value == "" {
		return Content.Language
	}

	langcode := Content.Language

	prefs, err := ParsePreferences(value)
	// If language parsing fails, continue with request.
	// See https://tools.ietf.org/html/rfc7231#section-5.3.5
	if err == nil {
		// If langcode is not listed, give it a competitive value for sanity.
		// The value most likely is still "en" (English).
		if _, ok := prefs[langcode]; !ok {
			prefs[langcode] = 0.85
		}
		for code, value := range prefs {
			if value > prefs[langcode] {
				langcode = code
			}
		}
	}
	return langcode
}

func init() {
	// Set content defaults
	Content.Mediatype = defaultMediatype
	Content.Version = defaultVersion
	Content.Language = defaultLanguage

	// just in case
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".xml", "application/xml")
}
