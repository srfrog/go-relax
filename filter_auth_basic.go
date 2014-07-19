// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

// FilterAuthBasic is a Filter that implements HTTP Basic Authentication as
// described in http://www.ietf.org/rfc/rfc2617.txt
type FilterAuthBasic struct {
	// Realm is the authentication realm.
	// This defaults to "Authorization Required"
	Realm string

	// Authenticate is a function that will perform the actual authentication
	// check.
	// It should expect a username and password, then return true if those
	// credentials are accepted; false otherwise.
	// If no function is assigned, it defaults to a function that denies all
	// (false).
	Authenticate func(string, string) bool
}

// ErrAuthInvalidRequest is returned when the auth request don't match the expected
// challenge.
var ErrAuthInvalidRequest = errors.New("Auth: Invalid authorization request")

// ErrAuthInvalidSyntax is returned when the syntax of the credentials is not what is
// expected.
var ErrAuthInvalidSyntax = errors.New("Auth: Invalid credentials syntax")

// denyAllAccess is the default Authenticate function, and as the name
// implies, will deny all access by returning false.
func denyAllAccess(username, password string) bool {
	return false
}

func getUserPass(header string) ([]string, error) {
	credentials := strings.Split(header, " ")
	if len(credentials) != 2 || credentials[0] != "Basic" {
		return nil, ErrAuthInvalidRequest
	}

	authstr, err := base64.StdEncoding.DecodeString(credentials[1])
	if err != nil {
		return nil, err
	}

	userpass := strings.Split(string(authstr), ":")
	if len(userpass) != 2 {
		return nil, ErrAuthInvalidSyntax
	}

	return userpass, nil
}

// Run runs the filter and passes down the following Info:
//		re.Info.Get("auth.user") // auth user
//		re.Info.Get("auth.type") // auth scheme type. e.g., "basic"
func (self *FilterAuthBasic) Run(next HandlerFunc) HandlerFunc {
	if self.Realm == "" {
		Log.Println(LOG_WARN, "FilterAuthBasic: using default realm")
		self.Realm = "Authorization Required"
	}
	self.Realm = strings.Replace(self.Realm, `"'`, "", -1)

	if self.Authenticate == nil {
		Log.Println(LOG_ALERT, "FilterAuthBasic: denying all access; no authenticate function set")
		self.Authenticate = denyAllAccess
	}

	return func(rw ResponseWriter, re *Request) {
		header := re.Header.Get("Authorization")
		if header == "" {
			MustAuthenticate(rw, "Basic realm=\""+self.Realm+"\"")
			return
		}

		userpass, err := getUserPass(header)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}

		if !self.Authenticate(userpass[0], userpass[1]) {
			MustAuthenticate(rw, "Basic realm=\""+self.Realm+"\"")
			return
		}

		re.Info.Set("auth.user", userpass[0])
		re.Info.Set("auth.type", "basic")

		next(rw, re)
	}
}

// MustAuthenticate is a helper function used to send the WWW-Authenticate
// HTTP header.
// challenge is the auth scheme and the realm, as specified in section 2 of
// RFC 2617.
func MustAuthenticate(w http.ResponseWriter, challenge string) {
	w.Header().Set("WWW-Authenticate", challenge)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
