// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

// StatusError is an error with a HTTP Status code. It allows errors to be
// RESTful and uniform.
type StatusError struct {
	// Code is meant for a HTTP status code or any other numeric ID.
	Code int `json:"code"`

	// Message is the default error message used in logs.
	Message string `json:"message"`

	// Details can be any data structure that gives more information about the
	// error.
	Details interface{} `json:"details,omitempty"`
}

// StatusError implements the error interface.
func (self *StatusError) Error() string { return self.Message }

// BUG(TODO): StatusError is too shallow, need to implement better error system with locale support.
// BUG(TODO): StatusError is also tied to JSON, it needs to support any encoding type.
//
