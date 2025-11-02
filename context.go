// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package relax

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"context"
)

// HandlerFunc is simply a version of http.HandlerFunc that uses Context.
// All filters must return and accept this type.
type HandlerFunc func(*Context)

// Context has information about the request and filters. It implements
// http.ResponseWriter.
type Context struct {
	context.Context

	// ResponseWriter is the response object passed from ``net/http``.
	http.ResponseWriter
	wroteHeader bool
	status      int
	bytes       int

	// Request points to the http.Request information for this request.
	Request *http.Request

	// PathValues contains the values matched in PSEs by the router. It is a
	// name=values map (map[string][]string).
	// Examples:
	//
	//		ctx.PathValues.Get("username") // returns the first value for "username"
	//		ctx.PathValues.Get("_2")       // values are also accessible by index
	//		ctx.PathValues["colors"]       // if more than one color value.
	//
	// See also: Router, url.Values
	PathValues url.Values

	// Encode is the media encoding function requested by the client.
	// To see the media type use:
	//
	//		ctx.Get("content.encoding")
	//
	// See also: Encoder.Encode
	Encode func(io.Writer, interface{}) error

	// Decode is the decoding function when this request was made. It expects an
	// object that implements io.Reader, usually Request.Body. Then it will decode
	// the data and try to save it into a variable interface.
	// To see the media type use:
	//
	//		ctx.Get("content.decoding")
	//
	// See also: Encoder.Decode
	Decode func(io.Reader, interface{}) error
}

// contextPool allows us to reuse some Context objects to conserve resources.
var contextPool = sync.Pool{
	New: func() interface{} { return new(Context) },
}

// newContext returns a new Context object.
// This function will alter Request.URL, adding scheme and host:port as provided by the client.
func newContext(parent context.Context, w http.ResponseWriter, r *http.Request) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.Context = parent
	ctx.ResponseWriter = w
	ctx.Request = r
	return ctx
}

// free frees a Context object back to the usage pool for later, to conserve
// system resources.
func (ctx *Context) free() {
	ctx.ResponseWriter = nil
	ctx.wroteHeader = false
	ctx.status = 0
	ctx.bytes = 0
	ctx.PathValues = nil
	ctx.Decode = nil
	ctx.Encode = nil
	contextPool.Put(ctx)
}

// Clone returns a shallow cloned context using 'w', an http.ResponseWriter object.
// If 'w' is nil, the ResponseWriter value can be assigned after cloning.
func (ctx *Context) Clone(w http.ResponseWriter) *Context {
	clone := contextPool.Get().(*Context)
	clone.Context = ctx.Context
	clone.ResponseWriter = w
	clone.Request = ctx.Request
	clone.PathValues = ctx.PathValues
	clone.bytes = ctx.bytes
	clone.Decode = ctx.Decode
	clone.Encode = ctx.Encode
	return clone
}

// Set stores the value of key in the Context k/v tree.
func (ctx *Context) Set(key string, value interface{}) {
	ctx.Context = context.WithValue(ctx.Context, key, value)
}

// Get retrieves the value of key from Context storage. The value is returned
// as an interface so it must be converted to an actual type. If the type implements
// fmt.Stringer then it may be used by functions that expect a string.
func (ctx *Context) Get(key string) interface{} {
	return ctx.Context.Value(key)
}

// Header implements ResponseWriter.Header
func (ctx *Context) Header() http.Header {
	return ctx.ResponseWriter.Header()
}

// Write implements ResponseWriter.Write
func (ctx *Context) Write(b []byte) (int, error) {
	n, err := ctx.ResponseWriter.Write(b)
	ctx.bytes += n
	return n, err
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

// Bytes returns the number of bytes written in the response.
func (ctx *Context) Bytes() int {
	return ctx.bytes
}

/*
Respond writes a response back to the client. A complete RESTful response
should be contained within a structure.

'v' is the object value to be encoded. 'code' is an optional HTTP status code.

If at any point the response fails (due to encoding or system issues), an
error is returned but not written back to the client.

	type Message struct {
		Status int    `json:"status"`
		Text   string `json:"text"`
	}

	ctx.Respond(&Message{Status: 201, Text: "Ticket created"}, http.StatusCreated)

See also: Context.Encode, WriteHeader
*/
func (ctx *Context) Respond(v interface{}, code ...int) error {
	if code != nil {
		ctx.WriteHeader(code[0])
	}
	err := ctx.Encode(ctx.ResponseWriter, v)
	if err != nil {
		// encoding failed, most likely we tried to encode something that hasn't
		// been made marshable yet.
		panic(err)
	}
	return err
}

/*
Error sends an error response, with appropriate encoding. It basically calls
Respond using a status code and wrapping the message in a StatusError object.

'code' is the HTTP status code of the error. 'message' is the actual error message
or reason. 'details' are additional details about this error (optional).

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
}

/*
Format implements the fmt.Formatter interface, based on Apache HTTP's
CustomLog directive. This allows a Context object to have Sprintf verbs for
its values. See: https://httpd.apache.org/docs/2.4/mod/mod_log_config.html#formats

	Verb	Description
	----	---------------------------------------------------

	%%  	Percent sign
	%a  	Client remote address
	%b  	Size of response in bytes, excluding headers. Or '-' if zero.
	%#a 	Proxy client address, or unknown.
	%h  	Remote hostname. Will perform lookup.
	%l  	Remote ident, will write '-' (only for Apache log support).
	%m  	Request method
	%q  	Request query string.
	%r  	Request line.
	%#r 	Request line without protocol.
	%s  	Response status code.
	%#s 	Response status code and text.
	%t  	Request time, as string.
	%u  	Remote user, if any.
	%v  	Request host name.
	%A  	User agent.
	%B  	Size of response in bytes, excluding headers.
	%D  	Time lapsed to serve request, in seconds.
	%H  	Request protocol.
	%I  	Bytes received.
	%L  	Request ID.
	%P  	Server port used.
	%R  	Referer.
	%U  	Request path.

Example:

	// Print request line and remote address.
	// Index [1] needed to reuse ctx argument.
	fmt.Printf("\"%r\" %[1]a", ctx)
	// Output:
	// "GET /v1/" 192.168.1.10
*/
func (ctx *Context) Format(f fmt.State, c rune) {
	var str string

	p, pok := f.Precision()
	if !pok {
		p = -1
	}

	switch c {
	case 'a':
		if f.Flag('#') {
			str = GetRealIP(ctx.Request)
			break
		}
		str = ctx.Request.RemoteAddr
	case 'b':
		if ctx.Bytes() == 0 {
			f.Write([]byte{45})
			return
		}
		fallthrough
	case 'B':
		str = strconv.Itoa(ctx.Bytes())
	case 'h':
		t := strings.Split(ctx.Request.RemoteAddr, ":")
		str = t[0]
	case 'l':
		f.Write([]byte{45})
		return
	case 'm':
		str = ctx.Request.Method
	case 'q':
		str = ctx.Request.URL.RawQuery
	case 'r':
		str = ctx.Request.Method + " " + ctx.Request.URL.RequestURI()
		if f.Flag('#') {
			break
		}
		str += " " + ctx.Request.Proto
	case 's':
		str = strconv.Itoa(ctx.Status())
		if f.Flag('#') {
			str += " " + http.StatusText(ctx.Status())
		}
	case 't':
		t := ctx.Get("request.start_time").(time.Time)
		str = t.Format("[02/Jan/2006:15:04:05 -0700]")
	case 'u':
		// XXX: i dont think net/http sets User
		if ctx.Request.URL.User == nil {
			f.Write([]byte{45})
			return
		}
		str = ctx.Request.URL.User.Username()
	case 'v':
		str = ctx.Request.Host
	case 'A':
		str = ctx.Request.UserAgent()
	case 'D':
		when := ctx.Get("request.start_time").(time.Time)
		if when.IsZero() {
			f.Write([]byte("%!(BADTIME)"))
			return
		}
		pok = false
		str = strconv.FormatFloat(time.Since(when).Seconds(), 'f', p, 32)
	case 'H':
		str = ctx.Request.Proto
	case 'I':
		str = fmt.Sprintf("%d", ctx.Request.ContentLength)
	case 'L':
		str = ctx.Get("request.id").(string)
	case 'P':
		s := strings.Split(ctx.Request.Host, ":")
		if len(s) > 1 {
			str = s[1]
			break
		}
		str = "80"
	case 'R':
		str = ctx.Request.Referer()
	case 'U':
		str = ctx.Request.URL.Path
	}
	if pok {
		str = str[:p]
	}
	f.Write([]byte(str))
}
