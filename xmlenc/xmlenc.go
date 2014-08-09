// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package xmlenc

import (
	"bytes"
	"encoding/xml"
	// "github.com/codehack/go-relax"
	".."
	"io"
	"io/ioutil"
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

// NewEncoderXML returns an EncoderXML object. This function will initiallize
// the object with sane defaults, for use with Service.encoders.
// Returns the new EncoderXML object.
func NewEncoderXML() *EncoderXML {
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
// Returns the XML content and nil on success, otherwise []byte{} and error
// on failure.
func (e *EncoderXML) Encode(v interface{}) ([]byte, error) {
	var bb bytes.Buffer
	var b []byte
	var err error

	if e.Indented {
		b, err = xml.MarshalIndent(v, "", "\t")
	} else {
		b, err = xml.Marshal(v)
	}
	if err != nil {
		return nil, err
	}
	_, err = bb.WriteString(xml.Header)
	if err != nil {
		return nil, err
	}
	_, err = bb.Write(b)
	if err != nil {
		return nil, err
	}

	return bb.Bytes(), nil
}

// Decode reads an XML payload (usually from Request.Body) and tries to
// set it to a variable v. If the payload is too large, with maximum
// EncoderXML.MaxBodySize, it will fail with error ErrBodyTooLarge
// Returns nil on success and error on failure.
func (e *EncoderXML) Decode(reader io.Reader, v interface{}) error {
	r := io.LimitReader(reader, e.MaxBodySize+1)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) > e.MaxBodySize {
		return relax.ErrBodyTooLarge
	}
	return xml.Unmarshal(b, v)
}
