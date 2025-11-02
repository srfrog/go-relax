// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package relax

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
)

// These status codes are inaccessible in net/http but they work with http.StatusText().
// They are included here as they might be useful.
// See: https://tools.ietf.org/html/rfc6585
const (
	// StatusUnprocessableEntity indicates the user sent content that while it is
	// syntactically correct, it might be erroneous.
	// See: http://tools.ietf.org/html/rfc4918#section-11.2
	StatusUnprocessableEntity = 422
	// StatusPreconditionRequired indicates that the origin server requires the
	// request to be conditional.
	StatusPreconditionRequired = 428
	// StatusTooManyRequests indicates that the user has sent too many requests
	// in a given amount of time ("rate limiting").
	StatusTooManyRequests = 429
	// StatusRequestHeaderFieldsTooLarge indicates that the server is unwilling to
	// process the request because its header fields are too large.
	StatusRequestHeaderFieldsTooLarge = 431
	// StatusNetworkAuthenticationRequired indicates that the client needs to
	// authenticate to gain network access.
	StatusNetworkAuthenticationRequired = 511
)

// NewRequestID returns a new request ID value based on UUID; or checks
// an id specified if it's valid for use as a request ID. If the id is not
// valid then it returns a new ID.
//
// A valid ID must be between 20 and 200 chars in length, and URL-encoded.
func NewRequestID(id string) string {
	if id == "" {
		return uuid.Must(uuid.NewV4()).String()
	}
	l := 0
	for i, c := range id {
		switch {
		case 'A' <= c && c <= 'Z':
		case 'a' <= c && c <= 'z':
		case '0' <= c && c <= '9':
		case c == '-', c == '_', c == '.', c == '~', c == '%', c == '+':
		case i > 199:
			fallthrough
		default:
			return uuid.Must(uuid.NewV4()).String()
		}
		l = i
	}
	if l < 20 {
		return uuid.Must(uuid.NewV4()).String()
	}
	return id
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
	prefs := make(map[string]float32)
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

// IsRequestSSL returns true if the request 'r' is done via SSL/TLS.
// SSL status is guessed from value of Request.TLS. It also checks the value
// of the X-Forwarded-Proto header, in case the request is proxied.
// Returns true if the request is via SSL, false otherwise.
func IsRequestSSL(r *http.Request) bool {
	return (r.TLS != nil || r.URL.Scheme == "https" || r.Header.Get("X-Forwarded-Proto") == "https")
}

// GetRealIP returns the client address if the request is proxied. This is
// a best-guess based on the headers sent. The function will check the following
// headers, in order, to find a proxied client: Forwarded, X-Forwarded-For and
// X-Real-IP.
// Returns the client address or "unknown".
func GetRealIP(r *http.Request) string {
	// check if the IP address is hidden behind a proxy request.
	// See http://tools.ietf.org/html/rfc7239
	if v := r.Header.Get("Forwarded"); v != "" {
		values := strings.Split(v, ",")
		if strings.HasPrefix(values[0], "for=") {
			value := strings.Trim(values[0][4:], `"][`)
			if value[0] != '_' {
				return value
			}
		}
	}

	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		values := strings.Split(v, ", ")
		if values[0] != "unknown" {
			return values[0]
		}
	}

	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}

	return "unknown"
}
