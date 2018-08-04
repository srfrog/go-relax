// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"net/url"
	"testing"
)

var testRouter = newRouter()

var testRoutes = []struct {
	Method string
	Path   string
}{
	{"GET", "/posts"},
	{"GET", "/posts/{uint:id}"},
	{"GET", "/posts/{uint:id}/links"},
	{"GET", "/posts/{word:tag}"},
	{"GET", "/posts/{word:tag}/{uint:uid}"},
}

func testHandler(ctx *Context) {}

var testRequests = []struct {
	Method string
	Path   string
	Must   bool
}{
	{"GET", "/posts", true},
	{"GET", "/posts/123", true},
	{"GET", "/posts/444/links", true},
	{"GET", "/posts/something", true},
	{"GET", "/posts/tagged/666", true},
}

func TestFindHandler(t *testing.T) {
	for i := range testRoutes {
		testRouter.AddRoute(testRoutes[i].Method, testRoutes[i].Path, testHandler)
	}

	for i := range testRequests {
		var v url.Values
		_, err := testRouter.FindHandler(testRequests[i].Method, testRequests[i].Path, &v)
		if testRequests[i].Must && err != nil {
			t.Error(testRequests[i].Method, testRequests[i].Path, err.Error())
		}
	}
}
