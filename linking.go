// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"fmt"
	"reflect"
	"strings"
)

/*
Link an HTTP header tag that represents a hypertext relation link. It implements
HTTP web links between resources that are not format specific.

For details see also,
Web Linking: :https://tools.ietf.org/html/rfc5988
Relations: http://www.iana.org/assignments/link-relations/link-relations.xhtml
Item and Collection Link Relations: http://tools.ietf.org/html/rfc6573
Versioning: https://tools.ietf.org/html/rfc5829
URI Template: http://tools.ietf.org/html/rfc6570
Media: http://www.w3.org/TR/css3-mediaqueries/

The field title* ``Titlex`` must be encoded as per RFC5987.
See: http://greenbytes.de/tech/webdav/rfc5988.html#RFC5987

Extension field ``Ext`` must be name lowercase and quoted-string value,
as needed.

Example:

	link := Link{
		URI: "/v1/schemas",
		Rel: "index",
		Ext: "priority=\"important\"",
		Title: "Definition of schemas",
		Titlex: "utf-8'es'\"Definición de esquemas\"",
		HrefLang: "en-US",
		Media: "screen, print",
		Type: "text/html;charset=utf-8",
	}

*/
type Link struct {
	URI      string `json:"href"`
	Rel      string `json:"rel"`
	Anchor   string `json:"anchor,omitempty"`
	Rev      string `json:"rev,omitempty"`
	HrefLang string `json:"hreflang,omitempty"`
	Media    string `json:"media,omitempty"`
	Title    string `json:"title,omitempty"`
	Titlex   string `json:"title*,omitempty"`
	Type     string `json:"type,omitempty"`
	Ext      string
}

// String returns a string representation of a Link object. Suitable for use
// in "Link" HTTP headers.
func (l *Link) String() string {
	link := fmt.Sprintf(`<%s>`, l.URI)
	e := reflect.ValueOf(l).Elem()
	for i, j := 1, e.NumField(); i < j; i++ {
		n, v := e.Type().Field(i).Name, e.Field(i).String()
		if n == "Rel" && v == "" {
			v = "alternate"
		}
		if v == "" {
			continue
		}
		if n == "Ext" {
			link += fmt.Sprintf(`; %s`, v)
			continue
		}
		if n == "Titlex" {
			link += fmt.Sprintf(`; title*=%s`, v)
			continue
		}
		link += fmt.Sprintf(`; %s=%q`, strings.ToLower(n), v)
	}
	return link
}

// LinkHeader returns a complete Link header value that can be plugged
// into http.Header().Add(). Use this when you don't need a Link object
// for your relation, just a header.
// uri is the URI of target.
// param is one or more name=value pairs for link values. if nil, will default
// to rel="alternate" (as per https://tools.ietf.org/html/rfc4287#section-4.2.7).
// Returns two strings: "Link","Link header spec"
func LinkHeader(uri string, param ...string) (string, string) {
	value := []string{fmt.Sprintf(`<%s>`, uri)}
	if param == nil {
		param = []string{`rel="alternate"`}
	}
	value = append(value, param...)
	return "Link", strings.Join(value, "; ")
}

// relationHandler is a filter that adds link relations to the response.
func (r *Resource) relationHandler(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		for _, link := range r.links {
			ctx.Header().Add("Link", link.String())
		}
		next(ctx)
	}
}

// NewLink inserts new link relation for a resource. If the relation already exists,
// determined by comparing URI and relation type, then it is replaced with the new one.
func (r *Resource) NewLink(link *Link) {
	for k, v := range r.links {
		if v.URI == link.URI && v.Rel == link.Rel {
			r.links[k] = link
			return
		}
	}
	r.links = append(r.links, link)
}
