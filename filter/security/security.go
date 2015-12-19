// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package security

import (
	"github.com/codehack/go-relax"
	"net/http"
)

const (
	securityUACheckErr    = "Request forbidden by security rules.\nPlease make sure your request has an User-Agent header."
	securityXFrameDefault = "SAMEORIGIN"
	securityHSTSDefault   = "max-age=31536000; includeSubDomains"
	securityCacheDefault  = "no-store, must-revalidate"
	securityPragmaDefault = "no-cache"
)

// Filter Security provides some security options and checks.
// Most of the options are HTTP headers sent back so that web clients can
// adjust their configuration.
// See https://www.owasp.org/index.php/List_of_useful_HTTP_headers
type Filter struct {
	// UACheckDisable if false, a check is done to see if the client sent a valid non-emtpy
	// User-Agent header with the request.
	// Defaults to false.
	UACheckDisable bool

	// UACheckErrMsg is the response body sent when a client fails User-Agent check.
	// Defaults to (taken from Heroku's UA check message):
	// 	"Request forbidden by security rules.\n" +
	// 	"Please make sure your request has an User-Agent header."
	UACheckErrMsg string

	// XFrameDisable if false, will send a X-Frame-Options header with the response,
	// using the value in XFrameOptions. X-Frame-Options provides clickjacking protection.
	// For details see https://www.owasp.org/index.php/Clickjacking
	// https://www.rfc-editor.org/rfc/rfc7034.txt
	// http://tools.ietf.org/html/draft-ietf-websec-x-frame-options-12
	// Defaults to false.
	XFrameDisable bool

	// XFrameOptions expected values are:
	//		"DENY"                // no rendering within a frame
	//		"SAMEORIGIN"          // no rendering if origin mismatch
	//		"ALLOW-FROM {origin}" // allow rendering if framed by frame loaded from {origin};
	//			              // where {origin} is a top-level URL. ie., http//codehack.com
	// Only one value can be used at a time.
	// Defaults to "SAMEORIGIN"
	XFrameOptions string

	// XCTODisable if false, will send a X-Content-Type-Options header with the response
	// using the value "nosniff". This prevents Internet Explorer and Google Chrome from
	// MIME-sniffing and ignoring the value set in Content-Type.
	// Defaults to false.
	XCTODisable bool

	// HSTSDisable if false, will send a Strict-Transport-Security (HSTS) header
	// with the respose, using the value in HSTSOptions. HSTS enforces secure
	// connections to the server. http://tools.ietf.org/html/rfc6797
	// If the server is not on a secure HTTPS/TLS connection, it will temporarily
	// change to true.
	// Defaults to false.
	HSTSDisable bool

	// HSTSOptions are the values sent in an HSTS header.
	// Expected values are one or both of:
	//		"max-age=delta"     // delta in seconds, the time this host is a known HSTS host
	//		"includeSubDomains" // HSTS policy applies to this domain and all subdomains.
	// Defaults to "max-age=31536000; includeSubDomains"
	HSTSOptions string

	// CacheDisable if false, will send a Cache-Control header with the response,
	// using the value in CacheOptions. If this value is true, it will also
	// disable Pragma header (see below).
	// Defaults to false.
	CacheDisable bool

	// CacheOptions are the value sent in an Cache-Control header.
	// For details, see http://tools.ietf.org/html/rfc7234#section-5.2
	// Defaults to "no-store, must-revalidate"
	CacheOptions string

	// PragmaDisable if false and CacheDisable is false, will send a Pragma header
	// with the response, using the value "no-cache".
	// For details see http://tools.ietf.org/html/rfc7234#section-5.4
	// Defaults to false.
	PragmaDisable bool
}

// Run runs the filter.
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.UACheckErrMsg == "" {
		f.UACheckErrMsg = securityUACheckErr
	}
	if f.XFrameOptions == "" {
		f.XFrameOptions = securityXFrameDefault
	}
	if f.HSTSOptions == "" {
		f.HSTSOptions = securityHSTSDefault
	}
	if f.CacheOptions == "" {
		f.CacheOptions = securityCacheDefault
	}
	return func(ctx *relax.Context) {
		if !f.UACheckDisable {
			ua := ctx.Request.UserAgent()
			if ua == "" || ua == "Go 1.1 package http" {
				ctx.Error(http.StatusForbidden, f.UACheckErrMsg)
				return
			}
		}

		if !f.XCTODisable {
			ctx.Header().Set("X-Content-Type-Options", "nosniff")
		}

		if !f.XFrameDisable {
			ctx.Header().Set("X-Frame-Options", f.XFrameOptions)
		}

		// turn off HSTS if not on secure connection.
		if !f.HSTSDisable && relax.IsRequestSSL(ctx.Request) {
			ctx.Header().Set("Strict-Transport-Security", f.HSTSOptions)
		}

		if !f.CacheDisable {
			ctx.Header().Set("Cache-Control", f.CacheOptions)
			if !f.PragmaDisable {
				ctx.Header().Set("Pragma", "no-cache")
			}
		}

		next(ctx)
	}
}
