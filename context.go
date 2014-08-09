// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"github.com/codehack/go-environ"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// These status codes are inaccessible in net/http but they work with http.StatusText().
// They are included here as they might be useful.
// See also, https://tools.ietf.org/html/rfc6585
const (
	StatusPreconditionRequired          = 428
	StatusTooManyRequests               = 429
	StatusRequestHeaderFieldsTooLarge   = 431
	StatusNetworkAuthenticationRequired = 511
)

// Context has information about the request and filters. It implements
// http.ResponseWriter.
type Context struct {
	// ResponseWriter is the response object passed from ``net/http``.
	http.ResponseWriter
	wroteHeader bool
	status      int

	// Buffer points to a buffered context, started with Context.Capture.
	// If not capturing, Buffer is nil.
	// See also: ResponseBuffer
	Buffer *ResponseBuffer

	// Request points to the http.Request information for this request.
	Request *http.Request

	// PathValues contains the values matched in PSEs by the router. It is a
	// name=values map (map[string][]string).
	// Examples:
	//		ctx.PathValues.Get("username") // returns the first value for "username"
	//		ctx.PathValues.Get("_2")       // values are also accessible by index
	//		ctx.PathValues["colors"]       // if more than one color value.
	//
	// See also: Router, url.Values
	PathValues url.Values

	// Info contains information passed down from processed filters.
	// To print all values to stdout use:
	//		ctx.Info.Print()
	//
	// For usage, see http://github.com/codehack/go-environ
	Info *environ.Env

	// Encode is the media encoding function requested by the client.
	// To see the media type use:
	//		ctx.Info.Get("content.encoding")
	//
	// See also: Encoder.Encode
	Encode func(interface{}) ([]byte, error)

	// Decode is the decoding function when this request was made. It expects an
	// object that implements io.Reader, usually Request.Body. Then it will decode
	// the data and try to save it into a variable interface.
	// To see the media type use:
	//		ctx.Info.Get("content.decoding")
	//
	// See also: Encoder.Decode
	Decode func(io.Reader, interface{}) error
}

// contextPool allows us to reuse some Context objects to conserve resources.
var contextPool = sync.Pool{
	New: func() interface{} { return new(Context) },
}

// NewContext returns a new Context object.
// This function will alter Request.URL, adding scheme and host:port as provided by the client.
func NewContext(w http.ResponseWriter, r *http.Request) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.ResponseWriter = w
	ctx.Request = r
	ctx.Info = environ.NewEnv()

	// this little hack to make net/url work with full URLs.
	// net/http doesn't fill these for server requests, but we need them.
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
		if ctx.IsSSL() {
			r.URL.Scheme += "s"
		}
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}

	return ctx
}

// Free frees a Context object back to the usage pool for later, to conserve
// system resources.
func (ctx *Context) Free() {
	ctx.ResponseWriter = nil
	ctx.wroteHeader = false
	ctx.status = 0
	if ctx.Buffer != nil {
		ctx.Buffer.Free()
		ctx.Buffer = nil
	}
	ctx.Request = nil
	ctx.PathValues = nil
	ctx.Info.Free()
	ctx.Decode = nil
	ctx.Encode = nil
	contextPool.Put(ctx)
}

// Clone returns a shallow cloned context using 'w', an http.ResponseWriter object.
// If 'w' is nil, the ResponseWriter value can be assigned after cloning.
func (ctx *Context) Clone(w http.ResponseWriter) *Context {
	clone := NewContext(w, ctx.Request)
	clone.PathValues = ctx.PathValues
	clone.Info = ctx.Info
	clone.Decode = ctx.Decode
	clone.Encode = ctx.Encode
	return clone
}

// Capture starts a buffered context. All writes are diverted to a ResponseBuffer.
// Capture expects a call to Context.Release to end capturing.
// Returns a new buffered Context.
// See also: NewResponseBuffer, Context.Release
func (ctx *Context) Capture() *Context {
	ctx.Buffer = NewResponseBuffer(ctx.ResponseWriter)
	return ctx.Clone(ctx.Buffer)
}

// Release ends capturing within the context. Every Capture call needs
// a Release, otherwise the buffer will over-extend and the response will fail.
// You may or not defer this call after Capture, it depends on your state.
func (ctx *Context) Release() {
	if ctx.Buffer != nil {
		ctx.Buffer.Flush(ctx)
		ctx.Buffer = nil
	}
}

// IsSSL returns true if the context request is done via SSL/TLS.
// SSL status is guessed from value of Request.TLS. It also checks the value
// of the X-Forwarded-Proto header, in case the request is proxied.
func (ctx *Context) IsSSL() bool {
	return (ctx.Request.TLS != nil || ctx.Request.URL.Scheme == "https" || ctx.Request.Header.Get("X-Forwarded-Proto") == "https")
}

// ProxyClient returns the client address if the request is proxied. This is
// a best-guess based on the headers sent. The function will check the following
// headers, in order, to find a proxied client: Forwarded, X-Forwarded-For and
// X-Real-IP.
// Returns the client address or "unknown".
func (ctx *Context) ProxyClient() string {
	client := ctx.Info.Get("proxy_client")
	if client != "" {
		return client
	}

	// check if the IP address is hidden behind a proxy request.
	switch {
	default:
		// See http://tools.ietf.org/html/rfc7239
		if v := ctx.Request.Header.Get("Forwarded"); v != "" {
			values := strings.Split(v, ",")
			if strings.HasPrefix(values[0], "for=") {
				value := strings.Trim(values[0][4:], `"][`)
				if value[0] != '_' {
					client = value
					break
				}
			}
		}

		if v := ctx.Request.Header.Get("X-Forwarded-For"); v != "" {
			values := strings.Split(v, ", ")
			if values[0] != "unknown" {
				client = values[0]
				break
			}
		}

		if v := ctx.Request.Header.Get("X-Real-IP"); v != "" {
			client = v
			break
		}

		client = "unknown"
	}
	ctx.Info.Set("proxy_client", client)
	return client
}

// Header implements ResponseWriter.Header
func (ctx *Context) Header() http.Header {
	return ctx.ResponseWriter.Header()
}

// Write implements ResponseWriter.Write
func (ctx *Context) Write(b []byte) (int, error) {
	return ctx.ResponseWriter.Write(b)
}

// WriteHeader will force a status code header, if one hasn't been set.
// If no call to WriteHeader is done within this context, it defaults to
// http.StatusOK (200), which is sent by net/http.
func (ctx *Context) WriteHeader(code int) {
	if ctx.wroteHeader {
		return
	}
	ctx.wroteHeader = true
	ctx.status = code
	ctx.ResponseWriter.WriteHeader(code)
}

// Status returns the current known HTTP status code, or http.StatusOK if unknown.
func (ctx *Context) Status() int {
	if !ctx.wroteHeader {
		return http.StatusOK
	}
	return ctx.status
}

/*
Respond writes a response back to the client. A complete RESTful response
should be contained within a structure.

v is the object value to be encoded.

code is an optional HTTP status code.

If at any point the response fails (due to encoding or system issues), an
error is returned but not written back.

	type Message struct {
		Status string `json:"status"`
		Text   string `json:"text"`
	} `json:"apimessage"`

	ctx.Respond(&Message{Status: 201, Text: "Ticket created"}, http.StatusCreated)

See also: Context.Encode, Write, WriteHeader
*/
func (ctx *Context) Respond(v interface{}, code ...int) error {
	b, err := ctx.Encode(v)
	if err != nil {
		// encoding failed, most likely we tried to encode something that hasn't
		// been made marshable yet.
		Log.Println(LOG_ALERT, "Response encoding failed:", err.Error())
		// send a generic response because we can't send the real one.
		http.Error(ctx, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}
	if code != nil {
		ctx.WriteHeader(code[0])
	}
	_, err = ctx.Write(b)
	if err != nil {
		Log.Println(LOG_ALERT, "Response failed:", err.Error())
	}
	return err
}

/*
Error sends an error response, with appropiate encoding. It basically calls
Respond using a status code and wrapping the message in a StatusError object.

code is the HTTP status code of the error.

message is the actual error message or reason.

details are additional details about this error.

	type RouteDetails struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	ctx.Error(http.StatusNotImplemented, "That route is not implemented", &RouteDetails{"PATCH", "/v1/tickets/{id}"})

See also: Respond, StatusError
*/
func (ctx *Context) Error(code int, message string, details ...interface{}) {
	response := &StatusError{code, message, nil}
	if details != nil {
		response.Details = details[0]
	}
	ctx.Respond(response, code)
	Log.Println(LOG_DEBUG, "Error response:", code, "=>", message)
}
