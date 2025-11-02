// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package xmlenc

import (
	"encoding/xml"
	"io"

	"github.com/srfrog/go-relax"
)

// EncoderXML implements the relax.Encoder interface. It encode/decodes XML.
type EncoderXML struct {
	// MaxBodySize is the maximum size (in bytes) of XML content to be read (io.Reader)
	// Defaults to 4194304 (4MB)
	MaxBodySize int64

	// Indented indicates whether or not to output indented XML.
	// Defaults to false
	Indented bool

	// AcceptHeader is the media type used in Accept HTTP header.
	// Defaults to "application/xml"
	AcceptHeader string

	// ContentTypeHeader is the media type used in Content-Type HTTP header
	// Defaults to "application/xml;charset=utf-8"
	ContentTypeHeader string
}

// NewEncoder returns an EncoderXML object. This function will initiallize
// the object with sane defaults, for use with Service.encoders.
// Returns the new EncoderXML object.
func NewEncoder() *EncoderXML {
	return &EncoderXML{
		MaxBodySize:       4194304, // 4MB
		Indented:          false,
		AcceptHeader:      "application/xml",
		ContentTypeHeader: "application/xml;charset=utf-8",
	}
}

// Accept returns the media type for XML content, used in Accept header.
func (e *EncoderXML) Accept() string {
	return e.AcceptHeader
}

// ContentType returns the media type for XML content, used in the
// Content-Type header.
func (e *EncoderXML) ContentType() string {
	return e.ContentTypeHeader
}

// Encode will try to encode the value of v into XML. If EncoderJSON.Indented
// is true, then the XML will be indented with tabs.
// Returns the nil on success, and error on failure.
func (e *EncoderXML) Encode(writer io.Writer, v interface{}) error {
	_, err := writer.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	enc := xml.NewEncoder(writer)
	if e.Indented {
		enc.Indent("", "\t")
	}
	return enc.Encode(v)
}

// Decode reads an XML payload (usually from Request.Body) and tries to
// set it to a variable v. If the payload is too large, with maximum
// EncoderXML.MaxBodySize, it will fail with error ErrBodyTooLarge
// Returns nil on success and error on failure.
func (e *EncoderXML) Decode(reader io.Reader, v interface{}) error {
	r := &io.LimitedReader{R: reader, N: e.MaxBodySize}
	err := xml.NewDecoder(r).Decode(v)
	if err != nil && r.N == 0 {
		return relax.ErrBodyTooLarge
	}
	return err
}
