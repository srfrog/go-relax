// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"fmt"
	"reflect"
	"strings"
)

// Link represents a hypertext relation link. It implements HTTP web links
// between resources that are not format specific. For details see
// https://tools.ietf.org/html/rfc5988
// http://tools.ietf.org/html/draft-nottingham-linked-cache-inv-04
type Link struct {
	URI      string `json:"href"`
	Rel      string `json:"rel"`
	Anchor   string `json:"anchor,omitempty"`
	Rev      string `json:"rev,omitempty"`
	HrefLang string `json:"hreflang,omitempty"`
	Media    string `json:"media,omitempty"`
	Title    string `json:"title,omitempty"`
	Type     string `json:"type,omitempty"`
}

// String returns a string representation of a Link object. Suitable for use
// in Link: headers.
func (self *Link) String() string {
	link := fmt.Sprintf(`<%s>`, self.URI)
	e := reflect.ValueOf(self).Elem()
	for i := 1; i < e.NumField(); i++ {
		n, v := e.Type().Field(i).Name, e.Field(i).String()
		if v == "" {
			if n != "Rel" {
				continue
			}
			v = "alternate"
		}
		link += fmt.Sprintf(`; %s=%q`, strings.ToLower(n), v)
	}
	return link
}

// LinkHeader returns a complete Link: header value that can be plugged
// into http.Header().Add(). Use this when you don't need a Link object
// for your relation, just a header.
// uri is the URI of target
// param is one or more name=value pairs for link values. if nil, will default
// to rel="alternate" (as per RFC 4287).
// Returns two strings: "Link","Link header spec"
func LinkHeader(uri string, param ...string) (string, string) {
	value := []string{fmt.Sprintf(`<%s>`, uri)}
	if param == nil {
		param = []string{`rel="alternate"`}
	}
	value = append(value, param...)
	return "Link", strings.Join(value, "; ")
}
