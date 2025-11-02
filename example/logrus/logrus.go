// Copyright (c) 2025 srfrog - https://srfrog.dev
// Use of this source code is governed by the license in the LICENSE file.

// This is an example showing how to integrate logrus package with Relax.

package main

import (
	"github.com/sirupsen/logrus"
	"github.com/srfrog/go-relax"
	"github.com/srfrog/go-relax/filter/logs"
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
