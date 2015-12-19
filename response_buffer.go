// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"bytes"
	"io"
	"net/http"
	"sync"
)

/*
ResponseBuffer implements http.ResponseWriter, but redirects all
writes and headers to a buffer. This allows to inspect the response before
sending it. When a response is buffered, it needs an explicit call to
Flush or WriteTo to send it.

ResponseBuffer also implements io.WriteTo to write data to any object that
implements io.Writer.
*/
type ResponseBuffer struct {
	bytes.Buffer
	wroteHeader bool
	status      int
	header      http.Header
}

// Header returns the buffered header map.
func (rb *ResponseBuffer) Header() http.Header {
	return rb.header
}

// Write writes the data to the buffer.
// Returns the number of bytes written or error on failure.
func (rb *ResponseBuffer) Write(b []byte) (int, error) {
	return rb.Buffer.Write(b)
}

// WriteHeader stores the value of status code.
func (rb *ResponseBuffer) WriteHeader(code int) {
	if rb.wroteHeader {
		return
	}
	rb.wroteHeader = true
	rb.status = code
}

// Status returns the last known status code saved. If no status has been set,
// it returns http.StatusOK which is the default in ``net/http``.
func (rb *ResponseBuffer) Status() int {
	if rb.wroteHeader {
		return rb.status
	}
	return http.StatusOK
}

// WriteTo implements io.WriterTo. It sends the buffer, except headers,
// to any object that implements io.Writer. The buffer will be empty after
// this call.
// Returns the number of bytes written or error on failure.
func (rb *ResponseBuffer) WriteTo(w io.Writer) (int64, error) {
	return rb.Buffer.WriteTo(w)
}

// FlushHeader sends the buffered headers and status, but not the content, to
// 'w' an object that implements http.ResponseWriter.
// This function won't free the buffer or reset the headers but it will send
// the status using ResponseWriter.WriterHeader, if status was saved before.
// See also: ResponseBuffer.Flush, ResponseBuffer.WriteHeader
func (rb *ResponseBuffer) FlushHeader(w http.ResponseWriter) {
	for k, v := range rb.header {
		w.Header()[k] = v
	}
	if rb.wroteHeader {
		w.WriteHeader(rb.status)
	}
}

// Flush sends the headers, status and buffered content to 'w', an
// http.ResponseWriter object. The ResponseBuffer object is freed after this call.
// Returns the number of bytes written to 'w' or error on failure.
// See also: ResponseBuffer.Free, ResponseBuffer.FlushHeader, ResponseBuffer.WriteTo
func (rb *ResponseBuffer) Flush(w http.ResponseWriter) (int64, error) {
	defer rb.Free()
	rb.FlushHeader(w)
	return rb.WriteTo(w)
}

// reponseBufferPool allows us to reuse some ResponseBuffer objects to
// conserve system resources.
var reponseBufferPool = sync.Pool{
	New: func() interface{} { return new(ResponseBuffer) },
}

// NewResponseBuffer returns a ResponseBuffer object initialized with the headers
// of 'w', an object that implements ``http.ResponseWriter``.
// Objects returned using this function are pooled to save resources.
// See also: ResponseBuffer.Free
func NewResponseBuffer(w http.ResponseWriter) *ResponseBuffer {
	rb := reponseBufferPool.Get().(*ResponseBuffer)
	rb.header = make(http.Header, 0)
	for k, v := range w.Header() {
		rb.header[k] = v
	}
	return rb
}

// Free frees a ResponseBuffer object returning it back to the usage pool.
// Use with ``defer`` after calling NewResponseBuffer if WriteTo or Flush
// arent used. The values of the ResponseBuffer are reset and must be
// re-initialized.
func (rb *ResponseBuffer) Free() {
	rb.Buffer.Reset()
	rb.wroteHeader = false
	rb.status = 0
	rb.header = nil
	reponseBufferPool.Put(rb)
}
