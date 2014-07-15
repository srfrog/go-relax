// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

// HandlerFunc is simply a version of http.HandlerFunc that uses managed
// ResponseWriter and Request objects. All filters must return and accept
// this type.
type HandlerFunc func(ResponseWriter, *Request)

// All filters must implement the Filter interface.
// Filter functions are inter-connected functions that are executed in FIFO
// order. They are linked together via closures.
type Filter interface {
	// Run executes the current filter event in a chain.
	// It takes a HandlerFunc function argument, which is executed within the
	// closure returned.
	Run(HandlerFunc) HandlerFunc
}
