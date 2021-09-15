// Copyright 2014 Codehack http://codehack.com
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package logs

import (
	"log"
	"os"

	"github.com/srfrog/go-relax"
)

// Pre-made log formats. Most are based on Apache HTTP's.
// Note: the [n] notation will index an specific argument from Sprintf list.
const (
	// LogFormatRelax is the default Relax post-event format
	LogFormatRelax = "%s [%-.8[1]L] \"%#[1]r\" => \"%#[1]s\" done in %.6[1]Ds"

	// LogFormatCommon is similar to Apache HTTP's Common Log Format (CLF)
	LogFormatCommon = "%h %[1]l %[1]u %[1]t \"%[1]r\" %#[1]s %[1]b"

	// LogFormatExtended is similar to NCSA extended/combined log format
	LogFormatExtended = LogFormatCommon + " \"%[1]R\" \"%[1]A\""

	// LogFormatReferer is similar to Apache HTTP's Referer log format
	LogFormatReferer = "%R -> %[1]U"
)

/*
Filter Log provides pre- and post-request event logs. It uses a custom
log format similar to the one used for Apache HTTP CustomLog directive.

	myservice.Use(logrus.New())
	log := &log.Filter{Logger: myservice.Logger(), PreLogFormat: LogFormatReferer}
	log.Println("Filter implements Logger.")

	// Context-specific format verbs (see Context.Format)
	log.Panicf("Status is %s = bad status!", ctx)

*/
type Filter struct {
	// Logger is an interface that is based on Go's log package. Any logging
	// system that implements Logger can be used.
	// Defaults to the stdlog in 'log' package.
	relax.Logger

	// PreLogFormat is the format for the pre-request log entry.
	// Leave empty if no log even is needed.
	// Default to empty (no pre-log)
	PreLogFormat string

	// PostLogFormat is the format for the post-request log entry.
	// Defaults to the value of LogFormatRelax
	PostLogFormat string
}

// Run processes the filter. No info is passed.
func (f *Filter) Run(next relax.HandlerFunc) relax.HandlerFunc {
	if f.Logger == nil {
		f.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	if f.PostLogFormat == "" {
		f.PostLogFormat = LogFormatRelax
	}

	return func(ctx *relax.Context) {
		if f.PreLogFormat != "" {
			f.Printf(f.PreLogFormat, ctx)
		}

		next(ctx)

		f.Printf(f.PostLogFormat, ctx)
	}
}
