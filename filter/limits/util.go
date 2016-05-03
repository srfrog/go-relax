// Copyright 2014-present Codehack. All rights reserved. 
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package limits

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/codehack/go-relax"
)

// Min returns the smaller integer between a and b.
// If a is lesser than b it returns a, otherwise returns b.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MD5RequestKey returns a key made from MD5 hash of Request.RemoteAddr and
// Request.UserAgent.
func MD5RequestKey(c relax.Context) string {
	h := md5.New()
	host, _ := SplitPort(c.Request.RemoteAddr)
	h.Write([]byte(host))
	h.Write([]byte(c.Request.UserAgent()))
	return "quota:" + hex.EncodeToString(h.Sum(nil))
}

// SplitPort splits an host:port address and returns the parts.
func SplitPort(addr string) (string, string) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:]
		}
	}
	return addr, ""
}
