// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package xmlenc

import (
	"bytes"
	"encoding/xml"
	"github.com/codehack/go-relax"
	"io"
	"io/ioutil"
)

// EncoderXML implements the relax.Encoder interface. It encode/decodes XML.
type EncoderXML struct {
	// MaxBodySize is the maximum size (in bytes) of XML content to be read (io.Reader)
	MaxBodySize int64

	// Indented indicates whether or not to output indented XML.
	Indented bool
}

// Accept returns the MIME representation for xml content, used in Accept header.
func (_ *EncoderXML) Accept() string {
	return "application/xml"
}

// ContentType returns the MIME representation for xml content, used in the
// Content-Type header.
func (_ *EncoderXML) ContentType() string {
	return "application/xml;charset=utf-8"
}

// Encode converts a value v into its XML representation.
// Returns a byte slice of XML value, or error on failure.
// on failure.
func (self *EncoderXML) Encode(v interface{}) ([]byte, error) {
	var bb bytes.Buffer
	var b []byte
	var err error

	if self.Indented {
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
// set it to a variable v.
// Returns error on failure.
func (self *EncoderXML) Decode(reader io.Reader, v interface{}) error {
	if self.MaxBodySize == 0 {
		self.MaxBodySize = int64(1 << 21) // 2MB
	}
	r := io.LimitReader(reader, self.MaxBodySize+1)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if int64(len(b)) > self.MaxBodySize {
		return relax.ErrBodyTooLarge
	}
	return xml.Unmarshal(b, v)
}
