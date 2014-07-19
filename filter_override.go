// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"net/http"
)

// FilterOverrideMethods specifies the methods can be overriden.
// Format is FilterOverrideMethods["method"] = "override"
var FilterOverrideMethods = map[string]string{
	"DELETE":  "POST",
	"OPTIONS": "GET",
	"PATCH":   "POST",
	"PUT":     "POST",
}

// FilterOverride changes the Request.Method if the client specifies
// override via HTTP header or query. This allows clients with limited HTTP
// verbs to send REST requests through GET/POST.
type FilterOverride struct {
	// Header expected for HTTP Method override
	Header string

	// QueryVar is used if header can't be set
	QueryVar string
}

// Run runs the filter and passes down the following Info:
//		re.Info.Get("override.method") // method replaced. e.g., "PATCH"
func (self *FilterOverride) Run(next HandlerFunc) HandlerFunc {
	if self.Header == "" {
		self.Header = "X-HTTP-Method-Override"
	}
	if self.QueryVar == "" {
		self.QueryVar = "_method"
	}

	return func(rw ResponseWriter, re *Request) {
		if mo := re.URL.Query().Get(self.QueryVar); mo != "" {
			re.Header.Set(self.Header, mo)
		}
		if mo := re.Header.Get(self.Header); mo != "" {
			if mo != re.Method {
				override, ok := FilterOverrideMethods[mo]
				if !ok {
					rw.Error(http.StatusMethodNotAllowed, mo+" method is not overridable.")
					return
				}
				// check that the caller method matches the expected override. e.g., used GET for OPTIONS
				if re.Method != override {
					rw.Error(http.StatusPreconditionFailed, "must use "+override+" to override for "+mo)
					return
				}
				re.Method = override
				re.Header.Del(self.Header)
				re.Info.Set("override.method", override)
			}
		}
		next(rw, re)
	}
}
