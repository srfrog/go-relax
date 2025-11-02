// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package relax

import (
	"encoding/json"
	"errors"
	"io"
)

// ErrBodyTooLarge is returned by Encoder.Decode when the read length exceeds the
// maximum size set for payload.
var ErrBodyTooLarge = errors.New("encoder: Body too large")

/*
Encoder objects provide new data encoding formats.

Once a request enters service context, all responses are encoded according to the
assigned encoder. Relax includes support for JSON encoding. Other types of encoding
can be added by implementing the Encoder interface.
*/
type Encoder interface {
	// Accept returns the media type used in HTTP Accept header.
	Accept() string

	// ContentType returns the media type, and optionally character set,
	// for decoding used in Content-Type header.
	ContentType() string

	// Encode function encodes the value of an interface and writes it to an
	// io.Writer stream (usually an http.ResponseWriter object).
	Encode(io.Writer, interface{}) error

	// Decode function decodes input from an io.Reader (usually Request.Body) and
	// tries to save it to an interface variable.
	Decode(io.Reader, interface{}) error
}

// EncoderJSON implements the Encoder interface. It encode/decodes JSON data.
type EncoderJSON struct {
	// MaxBodySize is the maximum size (in bytes) of JSON payload to read.
	// Defaults to 2097152 (2MB)
	MaxBodySize int64

	// Indented indicates whether or not to output indented JSON.
	// Note: indented JSON is slower to encode.
	// Defaults to false
	Indented bool

	// AcceptHeader is the media type used in Accept HTTP header.
	// Defaults to "application/json"
	AcceptHeader string

	// ContentTypeHeader is the media type used in Content-Type HTTP header
	// Defaults to "application/json;charset=utf-8"
	ContentTypeHeader string
}

// NewEncoder returns an EncoderJSON object. This function will initiallize
// the object with sane defaults, for use with Service.encoders.
// Returns the new EncoderJSON object.
func NewEncoder() *EncoderJSON {
	return &EncoderJSON{
		MaxBodySize:       2097152, // 2MB
		Indented:          false,
		AcceptHeader:      "application/json",
		ContentTypeHeader: "application/json;charset=utf-8",
	}
}

// Accept returns the media type for JSON content, used in Accept header.
func (e *EncoderJSON) Accept() string {
	return e.AcceptHeader
}

// ContentType returns the media type for JSON content, used in the
// Content-Type header.
func (e *EncoderJSON) ContentType() string {
	return e.ContentTypeHeader
}

// Encode will try to encode the value of v into JSON. If EncoderJSON.Indented
// is true, then the JSON will be indented with tabs.
// Returns nil on success, error on failure.
func (e *EncoderJSON) Encode(writer io.Writer, v interface{}) error {
	if e.Indented {
		// indented is much slower...
		b, err := json.MarshalIndent(v, "", "\t")
		if err != nil {
			return err
		}
		_, err = writer.Write(b)
		return err
	}
	return json.NewEncoder(writer).Encode(v)
}

// Decode reads a JSON payload (usually from Request.Body) and tries to
// save it to a variable v. If the payload is too large, with maximum
// EncoderJSON.MaxBodySize, it will fail with error ErrBodyTooLarge
// Returns nil on success and error on failure.
func (e *EncoderJSON) Decode(reader io.Reader, v interface{}) error {
	r := &io.LimitedReader{R: reader, N: e.MaxBodySize}
	err := json.NewDecoder(r).Decode(v)
	if err != nil && r.N == 0 {
		return ErrBodyTooLarge
	}
	return err
}
