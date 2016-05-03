// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

// StatusError is an error with a HTTP Status code. It allows errors to be
// complete and uniform.
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
func (e *StatusError) Error() string { return e.Message }

// BUG(TODO): StatusError is too shallow, need to implement better error system with locale support.
