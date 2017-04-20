// Copyright 2017 Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package fail is used to manage Go errors as HTTP responses. The goal is to respond
to web clients nicely, and inspect errors.

To use fail, you must wrap Go errors using ``fail.Cause``. This function will
return a ``Fail`` object that implements the ``error`` interface. Also, the
location of the original error is saved in the object for later inspection.

The ``Fail`` object can be further handled with methods matching HTTP responses
such as ``fail.BadRequest`` and ``fail.NotFound``.

Finally, to respond to a web client we use the ``fail.Say`` function which returns
the HTTP status code and message that can be sent via ``http.Error``.
*/
package fail

// Version is the version of this package.
const Version = "0.0.1"
