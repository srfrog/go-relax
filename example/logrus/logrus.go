// Copyright 2014-present Codehack. All rights reserved.
// For mobile and web development visit http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// This is an example showing how to integrate logrus package with Relax.

package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/codehack/go-relax"
	"github.com/codehack/go-relax/filter/logs"
)

// HelloIndex just says Hello <whatever>
func HelloIndex(ctx *relax.Context) {
	ctx.Respond("Hello, " + ctx.PathValues.Get("_1"))
}

func main() {
	// log all service requests with standard log
	log1 := logrus.StandardLogger()
	svc := relax.NewService("/hello", &logs.Filter{Logger: log1})

	// the hello index also gets a json log
	log2 := logrus.New()
	log2.Formatter = new(logrus.JSONFormatter)
	svc.Root().GET("*", HelloIndex, &logs.Filter{Logger: log2})
	svc.Run()
}
