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

const defaultRealm = "Authorization Required"

// AuthBasic is a Filter that implements HTTP Basic Authentication as
// described in http://www.ietf.org/rfc/rfc2617.txt
type AuthBasic struct {
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

func denyAccess(username, password string) bool {
	return false
}

func getUserPass(header string) ([]string, error) {
	credentials := strings.Split(header, " ")
	if len(credentials) != 2 || credentials[0] != "Basic" {
		return nil, errors.New("Invalid authorization request")
	}

	authstr, err := base64.StdEncoding.DecodeString(credentials[1])
	if err != nil {
		return nil, err
	}

	userpass := strings.Split(string(authstr), ":")
	if len(userpass) != 2 {
		return nil, errors.New("Invalid credentials syntax")
	}

	return userpass, nil
}

func (self *AuthBasic) Run(next HandlerFunc) HandlerFunc {
	if self.Realm == "" {
		Log.Println(LOG_WARN, "AuthBasic: using default realm")
		self.Realm = defaultRealm
	}
	self.Realm = strings.Replace(self.Realm, `"'`, "", -1)

	if self.Authenticate == nil {
		Log.Println(LOG_ALERT, "AuthBasic: denying all access; no authenticate function set")
		self.Authenticate = denyAccess
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
