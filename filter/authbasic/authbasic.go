// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

package authbasic

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/srfrog/go-relax"
)

// Filter AuthBasic is a Filter that implements HTTP Basic Authentication as
// described in http://www.ietf.org/rfc/rfc2617.txt
type Filter struct {
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

// Errors returned by Filter AuthBasic that are general and could be reused.
var (
	// ErrAuthInvalidRequest is returned when the auth request don't match the expected
	// challenge.
	ErrAuthInvalidRequest = errors.New("auth: Invalid authorization request")

	// ErrAuthInvalidSyntax is returned when the syntax of the credentials is not what is
	// expected.
	ErrAuthInvalidSyntax = errors.New("auth: Invalid credentials syntax")
)

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
//
//	ctx.Get("auth.user") // auth user
//	ctx.Get("auth.type") // auth scheme type. e.g., "basic"
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Realm == "" {
		f.Realm = "Authorization Required"
	}
	f.Realm = strings.Replace(f.Realm, `"'`, "", -1)

	if f.Authenticate == nil {
		f.Authenticate = denyAllAccess
	}

	return func(ctx *relax.Context) {
		header := ctx.Request.Header.Get("Authorization")
		if header == "" {
			MustAuthenticate(ctx, "Basic realm=\""+f.Realm+"\"")
			return
		}

		userpass, err := getUserPass(header)
		if err != nil {
			http.Error(ctx, err.Error(), http.StatusBadRequest)
			return
		}

		if !f.Authenticate(userpass[0], userpass[1]) {
			MustAuthenticate(ctx, "Basic realm=\""+f.Realm+"\"")
			return
		}

		ctx.Set("auth.user", userpass[0])
		ctx.Set("auth.type", "basic")

		next(ctx)
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
