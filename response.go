// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"net/http"
)

const (
	// Status codes from WebDAV. See https://tools.ietf.org/html/rfc4918
	StatusUnprocessableEntity = 422
	StatusLocked              = 423
	StatusFailedDependency    = 424

	// Status codes in net/http but not accessible. See https://tools.ietf.org/html/rfc6585
	// These work with http.StatusText().
	StatusPreconditionRequired          = 428
	StatusTooManyRequests               = 429
	StatusRequestHeaderFieldsTooLarge   = 431
	StatusNetworkAuthenticationRequired = 511
)

// ResponseWriter extends the http.ResponseWriter interface to output RESTful
// responses. Objects that implement ResponseWriter are expected to handle
// content encoding accordingly.
type ResponseWriter interface {
	http.ResponseWriter

	// Respond sends a response, with proper content encoding.
	Respond(interface{}, ...int) error

	// Error sends an (encoded) error response with optional details.
	Error(int, string, ...interface{})

	// Status returns the known value of HTTP status code.
	Status() int
}

// responseWriter implements http.ResponseWriter and our managed ResponseWriter.
// wroteHeader is a boolean that is true if we sent a header through WriteHeader().
// status is the known status code sent via WriteHeader().
// Encode is the encoder function; it expects an object then returns its byte representation,
// or an error if failed.
type responseWriter struct {
	w           http.ResponseWriter
	wroteHeader bool
	status      int
	Encode      func(interface{}) ([]byte, error)
}

func (self *responseWriter) Header() http.Header {
	return self.w.Header()
}

func (self *responseWriter) WriteHeader(code int) {
	if self.wroteHeader {
		return
	}
	self.wroteHeader = true
	self.status = code
	self.w.WriteHeader(code)
}

func (self *responseWriter) Write(b []byte) (int, error) {
	return self.w.Write(b)
}

func (self *responseWriter) Status() int {
	if !self.wroteHeader {
		return http.StatusOK
	}
	return self.status
}

// Respond writes a response back to the client. A complete RESTful responses
// should be contained within a structure.
// v is the object value to be encoded.
// code is an optional HTTP status code.
// If at any point the response fails (due to encoding or system issues), the
// error is returned but not written back.
func (self *responseWriter) Respond(v interface{}, code ...int) error {
	b, err := self.Encode(v)
	if err != nil {
		// encoding failed, most likely we tried to encode something that hasn't
		// been made marshable yet.
		Log.Println(LOG_ALERT, "Response encoding failed:", err.Error())
		// send a generic response because we can't send the real one.
		http.Error(self, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}
	if code != nil {
		self.WriteHeader(code[0])
	}
	_, err = self.Write(b)
	if err != nil {
		Log.Println(LOG_ALERT, "Response failed:", err.Error())
	}
	return err
}

// Error sends an error response, with appropiate encoding.
// code is the HTTP status code of the error.
// message is the actual error message or reason.
// details are additional details about this error.
func (self *responseWriter) Error(code int, message string, details ...interface{}) {
	response := &StatusError{code, message, nil}
	if details != nil {
		response.Details = details[0]
	}
	self.Respond(response, code)
	Log.Println(LOG_DEBUG, "Error response:", code, "=>", message)
}
