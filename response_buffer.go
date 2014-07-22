// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

// responseBuffer implements http.ResponseWriter. It is used to redirect all
// writes and headers to a buffer, which can be written to a RewriteWriter.
type responseBuffer struct {
	ResponseWriter
	buf         bytes.Buffer
	wroteHeader bool
	status      int
	header      http.Header
}

// Header tracks header operations while buffering.
// FIXME: we might need to lookup header values before we started buffering..
func (self *responseBuffer) Header() http.Header {
	return self.header
}

// WriteHeader keeps the value of last status code while buffering.
func (self *responseBuffer) WriteHeader(code int) {
	if self.wroteHeader {
		return
	}
	self.wroteHeader = true
	self.status = code
}

// Write sends all content to the buffer. It wont be used unless
// Flush or WriteTo are called later.
func (self *responseBuffer) Write(b []byte) (int, error) {
	return self.buf.Write(b)
}

// Status returns the last known status code while buffering. If no status
// has been set, it returns the last known value from the caller ResponseWriter.
func (self *responseBuffer) Status() int {
	if self.wroteHeader {
		return self.status
	}
	return self.ResponseWriter.Status()
}

// WriteTo implements io.WriterTo. It sends all content, except headers,
// to any object that implements io.Writer. The headers are sent to
// the caller ResponseWriter. It will call Free() to return the buffer
// back to the pool.
// Returns the number of bytes written or error on failure.
func (self *responseBuffer) WriteTo(w io.Writer) (int64, error) {
	defer self.Free()
	self.FlushHeader()
	return self.buf.WriteTo(w)
}

// Flush send the contents of the buffer to the caller ResponseWriter.
func (self *responseBuffer) Flush() (int64, error) {
	return self.WriteTo(self.ResponseWriter)
}

// FlushHeader sends all the buffered headers, but no content.
// This function wont Free the buffer or reset the headers.
func (self *responseBuffer) FlushHeader() {
	for k, v := range self.header {
		self.ResponseWriter.Header()[k] = v
	}
	if self.wroteHeader {
		self.ResponseWriter.WriteHeader(self.status)
	}
}

// Len returns the number of bytes stored in the buffer.
func (self *responseBuffer) Len() int {
	return self.buf.Len()
}

// Bytes returns the contents of the buffer in a byte slice.
func (self *responseBuffer) Bytes() []byte {
	return self.buf.Bytes()
}

// reponseRewriterPool allows us to reuse some responseBuffer objects to
// conserve system resources.
var reponseRewriterPool = sync.Pool{
	New: func() interface{} { return new(responseBuffer) },
}

// Free returns a responseBuffer object back to the usage pool.
// Use with ``defer`` after calling NewResponseBuffer if WriteTo or Flush
// arent used.
func (self *responseBuffer) Free() {
	self.ResponseWriter = nil
	self.buf.Reset()
	self.wroteHeader = false
	self.status = 0
	self.header = nil
	reponseRewriterPool.Put(self)
}

// NewResponseBuffer create a responseBuffer object.
// responseBuffer is used to bypass writes to ResponseWriter using a bytes.Buffer.
// Returns new responseWriter object that uses a responseBuffer to bypass writes
// to ResponseWriter, and the responseBuffer object itself.
func NewResponseBuffer(rw ResponseWriter) (*responseWriter, *responseBuffer) {
	rr := reponseRewriterPool.Get().(*responseBuffer)
	rr.ResponseWriter = rw
	rr.header = make(http.Header)
	return &responseWriter{ResponseWriter: rr, Encode: rw.(*responseWriter).Encode}, rr
}
