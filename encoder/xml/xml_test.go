// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package xmlenc

import (
	"bytes"
	"encoding/xml"
	"testing"
)

type Object struct {
	XMLName xml.Name `xml:"object"`
	Name    string   `xml:"name"`
	Number  int      `xml:"number,attr"`
	Strings []string `xml:"strings>value"`
}

func TestEncoder(t *testing.T) {
	var bb bytes.Buffer

	xmlstr := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<object number="12345">
	<name>Full Name</name>
	<strings>
		<value>some</value>
		<value>strings</value>
		<value>here</value>
	</strings>
</object>`)

	reader := bytes.NewReader(xmlstr)
	object := &Object{}

	encoder := NewEncoder()
	encoder.Indented = true

	err := encoder.Decode(reader, object)
	if err != nil {
		t.Error(err.Error())
	}

	err = encoder.Encode(&bb, object)
	if err != nil {
		t.Error(err.Error())
	}
	if string(xmlstr) != bb.String() {
		t.Errorf("expected xmlstr but got something else.")
	}
}
