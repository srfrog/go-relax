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

// Objects that implement the Encoder interface provide encoding/decoding
// of content.
type Encoder interface {
	// Accept returns the MIME representation of encoding used in HTTP Accept header.
	Accept() string

	// ContentType returns the MIME representation of decoding used in Content-Type header.
	ContentType() string

	// Encode function encodes an interface variable into a byte slice value of its encoding.
	Encode(interface{}) ([]byte, error)

	// Decode function decodes input from a reader stream into an interface variable.
	Decode(io.Reader, interface{}) error
}

// EncoderJSON implements the Encoder interface. It encode/decodes JSON.
type EncoderJSON struct {
	// MaxBodySize is the maximum size (in bytes) of JSON content to be read (io.Reader)
	MaxBodySize int64

	// Indented indicates whether or not to output indented JSON.
	Indented bool
}

func (_ *EncoderJSON) Accept() string {
	return "application/json"
}

func (_ *EncoderJSON) ContentType() string {
	return "application/json;charset=utf-8"
}

func (self *EncoderJSON) Encode(v interface{}) ([]byte, error) {
	if self.Indented {
		return json.MarshalIndent(v, "", "\t")
	}
	return json.Marshal(v)
}

func (self *EncoderJSON) Decode(reader io.Reader, v interface{}) error {
	if self.MaxBodySize == 0 {
		self.MaxBodySize = int64(1 << 21) // 2MB
	}
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
