// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"net/http"
)

const (
	securityXFrameOptions                 = "SAMEORIGIN"
	securityStrictTransportSecurity       = "max-age=31536000; includeSubDomains"
	securityXPermittedCrossDomainPolicies = "master-only"
	securityXXSSProtection                = "1; mode=block"
	securityCacheControl                  = "no-store, must-revalidate"
	securityPragma                        = "no-cache"
)

// SecurityFilter is a Filter that provides some security options and checks.
// Most of the options are HTTP headers sent back so that web clients can
// adjust their configuration.
// See https://www.owasp.org/index.php/List_of_useful_HTTP_headers
type SecurityFilter struct {
	UACheckDisable bool
	UACheckErrMsg  string
	XFrameDisable  bool
	XFrameOptions  string
	CTSniffDisable bool
	HSTSODisable   bool
	HSTSOptions    string
	PCDPDisable    bool
	PCDPOptions    string
	XSSPDisable    bool
	XSSPOptions    string
	CacheDisable   bool
	CacheOptions   string
	PragmaDisable  bool
	PragmaOptions  string
}

// BUG(TODO): SecurityFilter need more docs about each option.

func (self *SecurityFilter) Run(next HandlerFunc) HandlerFunc {
	if self.UACheckErrMsg == "" {
		self.UACheckErrMsg = "Request forbidden by security rules.\n" +
			"Please make sure your request has a User-Agent header."
	}
	if self.XFrameOptions == "" {
		self.XFrameOptions = securityXFrameOptions
	}
	if self.HSTSOptions == "" {
		self.HSTSOptions = securityStrictTransportSecurity
	}
	if self.PCDPOptions == "" {
		self.PCDPOptions = securityXPermittedCrossDomainPolicies
	}
	if self.XSSPOptions == "" {
		self.XSSPOptions = securityXXSSProtection
	}
	if self.CacheOptions == "" {
		self.CacheOptions = securityCacheControl
	}
	if self.PragmaOptions == "" {
		self.PragmaOptions = securityPragma
	}

	return func(rw ResponseWriter, re *Request) {
		if !self.UACheckDisable {
			ua := re.UserAgent()
			if ua == "" || ua == "Go 1.1 package http" {
				rw.Error(http.StatusForbidden, self.UACheckErrMsg)
				return
			}
		}

		// This prevents Internet Explorer and Google Chrome from MIME-sniffing a
		// response away from the declared Content-Type header.
		if !self.CTSniffDisable {
			rw.Header().Set("X-Content-Type-Options", "nosniff")
		}

		// Clickjacking protection:
		// https://www.rfc-editor.org/rfc/rfc7034.txt
		// http://tools.ietf.org/html/draft-ietf-websec-x-frame-options-01
		if !self.XFrameDisable {
			rw.Header().Set("X-Frame-Options", self.XFrameOptions)
		}

		// http://tools.ietf.org/html/rfc6797
		if !self.HSTSODisable {
			rw.Header().Set("Strict-Transport-Security", self.HSTSOptions)
		}

		if !self.PCDPDisable {
			rw.Header().Set("X-Permitted-Cross-Domain-Policies", self.PCDPOptions)
		}

		if !self.XSSPDisable {
			rw.Header().Set("X-XSS-Protection", self.XSSPOptions)
		}

		if !self.CacheDisable {
			rw.Header().Set("Cache-Control", self.CacheOptions)
		}

		if !self.PragmaDisable {
			rw.Header().Set("Pragma", self.PragmaOptions)
		}

		next(rw, re)
	}
}
