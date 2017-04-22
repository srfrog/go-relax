// Copyright 2017 Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package fail

import (
	"fmt"
	"runtime"

	"net/http"
	"strconv"
	"strings"
)

const (
	textInternalServerError = "an internal error has occurred"
)

// ErrUnspecified is a fallback for fail without cause, or nil.
var ErrUnspecified = fmt.Errorf("unspecified error")

// Fail is an error that could be handled in an HTTP response.
// - Status: the HTTP Status code of the response (400-4XX, 500-5XX)
// - Message: friendly error message (for clients)
// - Details: slice of error details. e.g., form validation errors.
type Fail struct {
	Status  int      `json:"-"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	prev    error
	file    string
	line    int
}

// defaultFail is used with convenience functions.
var defaultFail = &Fail{}

// Cause wraps an error into a Fail that could be linked to another.
func Cause(prev error) *Fail {
	err := &Fail{
		prev: prev,
	}
	err.Caller(1)
	return err
}

// Error implements the error interface.
// Ideally, you don't want to send out this to web clients, this is meant to be
// used with logging and tools.
func (f *Fail) Error() string {
	if f.prev == nil {
		f.prev = ErrUnspecified
	}
	return fmt.Sprintf("%s:%d: %s", f.file, f.line, f.prev.Error())
}

// String implements the fmt.Stringer interface, to make fails errors print nicely.
func (f *Fail) String() string {
	return f.Message
}

/*
Format implements the fmt.Formatter interface. This allows a Fail object to have
Sprintf verbs for its values.

	Verb	Description
	----	---------------------------------------------------

	%%  	Percent sign
	%d		All fail details separated with commas (``Fail.Details``)
	%e		The original error (``error.Error``)
	%f		File name where the fail was called, minus the path.
	%l		Line of the file for the fail
	%m		The message of the fail (``Fail.Message``)
	%s		HTTP Status code (``Fail.Status``)

Example:

	// Print file, line, and original error.
	// Note: we use index [1] to reuse `f` argument.
	f := fail.Cause(err)
	fmt.Printf("%[1]f:%[1]l %[1]e", f)
	// Output:
	// alerts.go:123 missing argument to vars

*/
func (f *Fail) Format(s fmt.State, c rune) {
	var str string

	p, pok := s.Precision()
	if !pok {
		p = -1
	}

	switch c {
	case 'd':
		str = strings.Join(f.Details, ", ")
	case 'e':
		if f.prev == nil {
			str = ErrUnspecified.Error()
		} else {
			str = f.prev.Error()
		}
	case 'f':
		str = f.file
	case 'l':
		str = strconv.Itoa(f.line)
	case 'm':
		str = f.Message
	case 's':
		str = strconv.Itoa(f.Status)
	}
	if pok {
		str = str[:p]
	}
	s.Write([]byte(str))

}

// Caller finds the file and line where the failure happened.
// `skip` is the number of calls to skip, not including this call.
// If you use this from a point(s) which is not the error location, then that
// call must be skipped.
func (f *Fail) Caller(skip int) {
	_, file, line, _ := runtime.Caller(skip + 1)
	f.file = file[strings.LastIndex(file, "/")+1:]
	f.line = line
}

// BadRequest changes the Go error to a "Bad Request" fail.
// `m` is the reason why this is a bad request.
// `details` is an optional slice of details to explain the fail.
func (f *Fail) BadRequest(m string, details ...string) error {
	f.Status = http.StatusBadRequest
	f.Message = m
	f.Details = details
	return f
}

// BadRequest is a convenience function to return a Bad Request fail when there's
// no Go error.
func BadRequest(m string, fields ...string) error {
	return defaultFail.BadRequest(m, fields...)
}

// Forbidden changes an error to a "Forbidden" fail.
// `m` is the reason why this action is forbidden.
func (f *Fail) Forbidden(m string) error {
	f.Status = http.StatusForbidden
	f.Message = m
	return f
}

// Forbidden is a convenience function to return a Forbidden fail when there's
// no Go error.
func Forbidden(m string) error {
	return defaultFail.Forbidden(m)
}

// NotFound changes the error to an "Not Found" fail.
func (f *Fail) NotFound(m ...string) error {
	if m == nil {
		m = []string{"object not found"}
	}
	f.Status = http.StatusNotFound
	f.Message = m[0]
	return f
}

// NotFound is a convenience function to return a Not Found fail when there's
// no Go error.
func NotFound(m ...string) error {
	return defaultFail.NotFound(m...)
}

// Unauthorized changes the error to an "Unauthorized" fail.
func (f *Fail) Unauthorized(m string) error {
	f.Status = http.StatusUnauthorized
	f.Message = m
	return f
}

// Unauthorized is a convenience function to return an Unauthorized fail when there's
// no Go error.
func Unauthorized(m string) error {
	return defaultFail.Unauthorized(m)
}

// Unexpected morphs the error into an "Internal Server Error" fail.
func (f *Fail) Unexpected() error {
	f.Status = http.StatusInternalServerError
	f.Message = textInternalServerError
	return f
}

// Unexpected is a convenience function to return an Internal Server Error fail
// when there's no Go error.
func Unexpected() error {
	return defaultFail.Unexpected()
}

// Say returns the HTTP status and message response for a handled fail.
// If the error is nil, then there's no error -- say everything is OK.
// If the error is not a handled fail, then convert it to an unexpected fail.
func Say(err error) (int, string) {
	switch e := err.(type) {
	case nil:
		return http.StatusOK, "OK"
	case *Fail:
		return e.Status, e.Message
	}

	// handle this unhandled unknown error
	f := &Fail{
		Status:  http.StatusInternalServerError,
		Message: textInternalServerError,
		prev:    err,
	}
	f.Caller(2)

	return f.Status, f.Message
}

// IsBadRequest returns true if fail is a Bad Request fail, false otherwise.
func IsBadRequest(err error) bool {
	e, ok := err.(*Fail)
	return ok && e.Status == http.StatusBadRequest
}

// IsUnauthorized returns true if fail is a Unauthorized fail, false otherwise.
func IsUnauthorized(err error) bool {
	e, ok := err.(*Fail)
	return ok && e.Status == http.StatusUnauthorized
}

// IsForbidden returns true if fail is a Forbidden fail, false otherwise.
func IsForbidden(err error) bool {
	e, ok := err.(*Fail)
	return ok && e.Status == http.StatusForbidden
}

// IsNotFound returns true if fail is a Not Found fail, false otherwise.
func IsNotFound(err error) bool {
	e, ok := err.(*Fail)
	return ok && e.Status == http.StatusNotFound
}

// IsUnexpected returns true if fail is an internal fail, false otherwise.
// This type of fail might be coming from an unhandled source.
func IsUnexpected(err error) bool {
	e, ok := err.(*Fail)
	return ok && e.Status == http.StatusInternalServerError
}

// IsUnknown returns true if the fail is not handled through this interface,
// false otheriwse.
func IsUnknown(err error) bool {
	_, ok := err.(*Fail)
	return !ok
}
