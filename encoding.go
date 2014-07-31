// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
)

// ErrBodyTooLarge is returned when the read length exceeds the maximum size
// set for decoding payload.
var ErrBodyTooLarge = errors.New("Encoder: Body too large")

/*
Encoder

Objects that implement the Encoder interface provide new content encoding formats.

Once a request enters service context, all responses are encoded according to the
assigned encoder. Relax includes support for JSON encoding. Other types of encoding
can be added by implementing the Encoder interface.

In common, all requests may send an HTTP Accept header with the form:

	Accept: application/vnd.relax+{encoding-type}; version=XX; lang=YY

Where encoding-type is the short notation of the media encoding format, for
example, "json" or "xml". ``version`` is an optional string to the content version;
API version. ``lang`` is the preferred language notation in RFC 5646 format, en-US,
for example. version and lang are optional but encoding-type is not. If the Accept
header is not sent, then encoding is assumed to be satisfied by the encoder's
default representation, which is "application/json".

Info passed down from content negotiation with the Request.Info object:

	re.Info.Get("content.encoding") // MIME type used for encoding. e.g., "application/json"
	re.Info.Get("content.version") // requested version, or "current"
	re.Info.Get("content.language") // requested language, or "en_US"

See also, http://tools.ietf.org/html/rfc5646; tags to identify languages.
*/
type Encoder interface {
	// Accept returns the media type used in HTTP Accept header.
	Accept() string

	// ContentType returns the media type, and optionally character set,
	// for decoding used in Content-Type header.
	ContentType() string

	// Encode function encodes an interface variable into a byte slice value of its encoding.
	Encode(interface{}) ([]byte, error)

	// Decode function decodes input from a io.Reader (usually Request.Body) and
	// tries to save it to an interface variable.
	Decode(io.Reader, interface{}) error
}

// EncoderJSON implements the Encoder interface. It encode/decodes JSON content.
type EncoderJSON struct {
	// MaxBodySize is the maximum size (in bytes) of JSON content to be read (io.Reader)
	// Defaults to 2097152 (2MB)
	MaxBodySize int64

	// Indented indicates whether or not to output indented JSON.
	// Defaults to false
	Indented bool

	// AcceptHeader is the media type used in Accept HTTP header.
	// Defaults to "application/json"
	AcceptHeader string

	// ContentTypeHeader is the media type used in Content-Type HTTP header
	// Defaults to "application/json;charset=utf-8"
	ContentTypeHeader string
}

// NewEncoderJSON returns an EncoderJSON object. This function will initiallize
// the object with sane defaults, for use with Service.encoders.
// Returns the new EncoderJSON object.
func NewEncoderJSON() *EncoderJSON {
	return &EncoderJSON{
		MaxBodySize:       2097152, // 2MB
		Indented:          false,
		AcceptHeader:      "application/json",
		ContentTypeHeader: "application/json;charset=utf-8",
	}
}

// Accept returns the media type for JSON content, used in Accept header.
func (self *EncoderJSON) Accept() string {
	return self.AcceptHeader
}

// ContentType returns the media type for JSON content, used in the
// Content-Type header.
func (self *EncoderJSON) ContentType() string {
	return self.ContentTypeHeader
}

// Encode will try to encode the value of v into JSON. If EncoderJSON.Indented
// is true, then the JSON will be indented with tabs.
// Returns the JSON content and nil on success, otherwise []byte{} and error
// on failure.
func (self *EncoderJSON) Encode(v interface{}) ([]byte, error) {
	if self.Indented {
		return json.MarshalIndent(v, "", "\t")
	}
	return json.Marshal(v)
}

// Decode reads a JSON payload (usually from Request.Body) and tries to
// set it to a variable v. If the payload is too large, with maximum
// EncoderJSON.MaxBodySize, it will fail with error ErrBodyTooLarge
// Returns nil on success and error on failure.
func (self *EncoderJSON) Decode(reader io.Reader, v interface{}) error {
	r := io.LimitReader(reader, self.MaxBodySize+1)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) > self.MaxBodySize {
		return ErrBodyTooLarge
	}
	return json.Unmarshal(b, v)
}
