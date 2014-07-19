// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"io"
	"net/http"
	"sync"
)

// responseRewriter allows any object that implements io.Writer to bypass ResponseWriter.
type responseRewriter struct {
	io.Writer
	http.ResponseWriter
}

// Write sends output to io.Writer directly, bypassing ResponseWriter.
func (self *responseRewriter) Write(b []byte) (int, error) {
	n, err := self.Writer.Write(b)
	if err != nil {
		Log.Println(LOG_CRIT, "responseRewriter:", err.Error())
	}
	return n, err
}

// reponseRewriterPool allows us to reuse some responseRewriter objects to conserve resources.
var reponseRewriterPool = sync.Pool{
	New: func() interface{} { return new(responseRewriter) },
}

// Free returns a responseRewriter object back to pool.
// Use with ``defer`` after calling NewResponseRewriter()
func (self *responseRewriter) Free() {
	self.ResponseWriter = nil
	if closer, ok := self.Writer.(io.Closer); ok {
		closer.Close()
	}
	self.Writer = nil
	reponseRewriterPool.Put(self)
}

// NewResponseRewriter fetches a responseRewriter object from pool or returns a new one.
// responseRewriter is used to bypass writes to ResponseWriter and write to another
// object that implements io.Writer.
func NewResponseRewriter(w io.Writer, rw http.ResponseWriter) *responseRewriter {
	rr := reponseRewriterPool.Get().(*responseRewriter)
	rr.Writer = w
	rr.ResponseWriter = rw
	return rr
}
