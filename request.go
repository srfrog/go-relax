// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"github.com/codehack/go-environ"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// Request is an enhanced version of http.Request that includes path values,
// info from filters and the service decoder.
type Request struct {
	// Request points to the http.Request information for this request.
	*http.Request

	// PathValues contains the name/value pairs of routes matched.
	PathValues url.Values

	// Info contains information passed down from processed filters.
	// Use ``re.Info.Print()`` to print all values.
	// For usage, check http://github.com/codehack/go-environ
	Info *environ.Env

	// Decode is the decoding function when this request was made.
	// Decore expects an object that implements io.Reader, usually Request.Body.
	// Then it will decode the payload from the client and try to save it into
	// a variable interface.
	Decode func(io.Reader, interface{}) error
}

// requestPool allows us to reuse some Request objects to conserve resources.
var requestPool = sync.Pool{
	New: func() interface{} { return new(Request) },
}

// free frees a Request object back to requestPool for later (re-)use
func (self *Request) free() {
	self.Request = nil
	self.PathValues = nil
	self.Decode = nil
	self.Info.Free()
	requestPool.Put(self)
}

// BaseURI returns the absolute base URI of this request.
func (self *Request) BaseURI() string {
	url := self.URL.ResolveReference(self.URL)
	return url.String()
}

// newRequest returns a new Request object.
func newRequest(r *http.Request) *Request {
	re := requestPool.Get().(*Request)
	re.Request = r
	re.PathValues = make(url.Values)
	re.Info = environ.NewEnv()

	// this little hack to make net/url work with full URLs.
	// net/http doesn't fill these for server requests, but we need them.
	if re.URL.Scheme == "" {
		re.URL.Scheme = "http"
		if re.TLS != nil {
			re.URL.Scheme += "s"
		}
	}
	if re.URL.Host == "" {
		re.URL.Host = re.Host
	}

	return re
}
