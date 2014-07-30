// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// The Router interface can be used to replace default routing engine (trieRegexpRouter).
type Router interface {
	// FindHandler should match request parameters to an existing resource handler and
	// return it. If no match is found, it should return an StatusError error which will
	// be sent to the requester. The default errors ErrRouteNotFound and
	// ErrRouteBadMethod cover the default cases.
	FindHandler(*Request) (HandlerFunc, error)

	// AddRoute is used to create new routes to resources. It expects the HTTP method
	// (GET, POST, ...) followed by the resource path (service + resource + variables)
	// and the handler function.
	AddRoute(string, string, HandlerFunc)
}

// These are errors returned by the default routing engine. You are encouraged to
// reuse them with your own routing engine.
var (
	// Returned when the path searched didn't reach a resource handler.
	ErrRouteNotFound = &StatusError{http.StatusNotFound, "That route was not found.", nil}

	// The path did not match a given HTTP method.
	ErrRouteBadMethod = &StatusError{http.StatusNotImplemented, "That method is not supported", nil}
)

// pathRegexpCache is a cache of all compiled regexp's so they can be reused.
var pathRegexpCache = make(map[string]*regexp.Regexp, 0)

// trieRegexpRouter implements Router with a trie that can store regular expressions.
// root points to the top of the tree from which all routes are searched and matched.
type trieRegexpRouter struct {
	root *trieNode
}

// trieNode contains the routing information.
// handler, if not nil, points to the resource handler served by a specific route.
// numExp is non-zero if the current path segment has regexp links.
// depth is the path depth of the current segment; 0 == HTTP verb.
// links are the contiguous path segments.
//
// For example, given the following route and handler:
//		"GET /api/users/111" -> users.GetUser()
//        - the path segment links are ["GET", "api", "users", "111"]
//        - "GET" has depth=0 and "111" has depth=3
//        - suppose "111" might be matched via regexp, then "users".numExp > 0
//        - "111" segment will point to the handler users.GetUser()
type trieNode struct {
	handler HandlerFunc
	numExp  int
	depth   int
	links   map[string]*trieNode
}

// segmentExp compiles the pattern string into a regexp so it can used in a
// path segment match. This function will panic if the regexp compilation fails.
//
// BUG(TODO): trieRegexpRouter has no support for custom regexp's for PSE's yet.
func segmentExp(pattern string) *regexp.Regexp {
	// turn "*" => "{wild}"
	pattern = strings.Replace(pattern, "*", `{wild}`, -1)
	// any: catch-all pattern
	p := regexp.MustCompile(`\{\w+\}`).
		ReplaceAllStringFunc(pattern, func(m string) string {
		return fmt.Sprintf(`(?P<%s>.+)`, m[1:len(m)-1])
	})
	// word: matches an alphanumeric word, with underscores.
	p = regexp.MustCompile(`\{(?:word\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		return fmt.Sprintf(`(?P<%s>\w+)`, m[6:len(m)-1])
	})
	// date: matches a date as described in ISO 8601. see: https://en.wikipedia.org/wiki/ISO_8601
	// accepted values:
	// 	YYYY
	// 	YYYY-MM
	// 	YYYY-MM-DD
	// 	YYYY-MM-DDTHH
	// 	YYYY-MM-DDTHH:MM
	// 	YYYY-MM-DDTHH:MM:SS[.NN]
	// 	YYYY-MM-DDTHH:MM:SS[.NN]Z
	// 	YYYY-MM-DDTHH:MM:SS[.NN][+-]HH
	// 	YYYY-MM-DDTHH:MM:SS[.NN][+-]HH:MM
	//
	p = regexp.MustCompile(`\{(?:date\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		name := m[6 : len(m)-1]
		return fmt.Sprintf(`(?P<%s>(`+
			`(?P<%s_year>\d{4})([/-]?(?P<%s_mon>(0[1-9])|(1[012]))([/-]?(?P<%s_mday>(0[1-9])|([12]\d)|(3[01])))?)?`+
			`(?:T(?P<%s_hour>([01][0-9])|(?:2[0123]))(\:?(?P<%s_min>[0-5][0-9])(\:?(?P<%s_sec>[0-5][0-9]([\,\.]\d{1,10})?))?)?(?:Z|([\-+](?:([01][0-9])|(?:2[0123]))(\:?(?:[0-5][0-9]))?))?)?`+
			`))`, name, name, name, name, name, name, name)
	})
	// geo: geo location in decimal. See http://tools.ietf.org/html/rfc5870
	// accepted values:
	// 	lat,lon           (point)
	// 	lat,lon,alt       (3d point)
	// 	lag,lon;u=unc     (circle)
	// 	lat,lon,alt;u=unc (sphere)
	// 	lat,lon;crs=name  (point with coordinate reference system (CRS) value)
	p = regexp.MustCompile(`\{(?:geo\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		name := m[5 : len(m)-1]
		return fmt.Sprintf(`(?P<%s_lat>\-?\d+(\.\d+)?)[,;](?P<%s_lon>\-?\d+(\.\d+)?)([,;](?P<%s_alt>\-?\d+(\.\d+)?))?(((?:;crs=)(?P<%s_crs>[\w\-]+))?((?:;u=)(?P<%s_u>\-?\d+(\.\d+)?))?)?`, name, name, name, name, name)
	})
	// hex: matches a hexadecimal number (assume 32bit)
	// accepted value: 0xNN
	p = regexp.MustCompile(`\{(?:hex\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		return fmt.Sprintf(`(?P<%s>(?:0x)?[[:xdigit:]]+)`, m[5:len(m)-1])
	})
	// float: matches a floating-point number
	p = regexp.MustCompile(`\{(?:float\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		return fmt.Sprintf(`(?P<%s>[\-+]?\d+\.\d+)`, m[7:len(m)-1])
	})
	// uint: matches an unsigned integer number (assume 32bit)
	p = regexp.MustCompile(`\{(?:uint\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		return fmt.Sprintf(`(?P<%s>\d{1,10})`, m[6:len(m)-1])
	})
	// int: matches a signed integer number (assume 32bit)
	p = regexp.MustCompile(`\{(?:int\:)\w+\}`).
		ReplaceAllStringFunc(p, func(m string) string {
		return fmt.Sprintf(`(?P<%s>[-+]?\d{1,10})`, m[5:len(m)-1])
	})
	return regexp.MustCompile(p)
}

// AddRoute breaks a path into segments and inserts them in the tree. If a
// segment contains matching {}'s then it is tried as a regexp segment, otherwise it is
// treated as a regular string segment.
func (self *trieRegexpRouter) AddRoute(method, path string, handler HandlerFunc) {
	node := self.root
	pseg := strings.Split(method+strings.TrimRight(path, "/"), "/")
	for i := range pseg {
		if (strings.Contains(pseg[i], "{") && strings.Contains(pseg[i], "}")) || strings.Contains(pseg[i], "*") {
			if _, ok := pathRegexpCache[pseg[i]]; !ok {
				pathRegexpCache[pseg[i]] = segmentExp(pseg[i])
			}
			node.numExp += 1
		}
		if node.links[pseg[i]] == nil {
			if node.links == nil {
				node.links = make(map[string]*trieNode, 0)
			}
			node.links[pseg[i]] = &trieNode{depth: node.depth + 1}
		}
		node = node.links[pseg[i]]
	}
	node.handler = handler
	Log.Println(LOG_DEBUG, "New route:", method, path)
}

// matchSegment tries to match a path segment 'pseg' to the node's regexp links.
// This function will return any path values matched so they can be used in
// Request.PathValues.
func (self *trieNode) matchSegment(pseg string, depth int, values *url.Values) *trieNode {
	if self.numExp == 0 {
		return self.links[pseg]
	}
	for pexp := range self.links {
		rx := pathRegexpCache[pexp]
		if rx == nil {
			continue
		}
		// this prevents the matching to be side-tracked by smaller paths.
		if depth > self.links[pexp].depth && self.links[pexp].links == nil {
			continue
		}
		m := rx.FindStringSubmatch(pseg)
		if len(m) > 1 && m[0] == pseg {
			sub := rx.SubexpNames()
			for i, n := 1, len(*values)/2; i < len(m); i++ {
				_n := fmt.Sprintf("_%d", n+i)
				Log.Println(LOG_DEBUG, "[router] Path value:", _n, "=", m[i])
				(*values).Set(_n, m[i])
				if sub[i] != "" {
					Log.Println(LOG_DEBUG, "[router] Path value:", sub[i], "=", m[i])
					(*values).Add(sub[i], m[i])
				}
			}
			return self.links[pexp]
		}
	}
	return self.links[pseg]
}

// FindHandler returns a resource handler that matches the requested route; or
// StatusError error if none matched.
func (self *trieRegexpRouter) FindHandler(re *Request) (HandlerFunc, error) {
	method := re.Method
	if method == "HEAD" {
		method = "GET"
	}
	node := self.root
	pseg := strings.Split(method+strings.TrimRight(re.URL.Path, "/"), "/")
	slen := len(pseg)
	for i := range pseg {
		if node == nil {
			if i <= 1 {
				return nil, ErrRouteBadMethod
			}
			return nil, ErrRouteNotFound
		}
		node = node.matchSegment(pseg[i], slen, &re.PathValues)
	}

	if node == nil || node.handler == nil {
		return nil, ErrRouteNotFound
	}
	return node.handler, nil
}

// newRouter returns a new trieRegexpRouter object with an initialized tree.
func newRouter() *trieRegexpRouter {
	return &trieRegexpRouter{new(trieNode)}
}
