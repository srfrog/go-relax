// Copyright 2014 Codehack.com All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package relax

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Pre-made log formats. Most are based on Apache HTTP's.
// Note: the [n] notation will index an specific argument from Sprintf list.
const (
	// LogFormatRelax is the default Relax post-event format
	LogFormatRelax = "%C [%-.8[1]L] \"%#[1]r\" => \"%#[1]s\" done in %.6[1]Ds"

	// LogFormatCommon is similar to Apache HTTP's Common Log Format (CLF)
	LogFormatCommon = "%h %[1]l %[1]u %[1]t \"%[1]r\" %#[1]s %[1]b"

	// LogFormatExtended is similar to NCSA extended/combined log format
	LogFormatExtended = LogFormatCommon + " \"%[1]R\" \"%[1]A\""

	// LogFormatReferer is similar to Apache HTTP's Referer log format
	LogFormatReferer = "%R -> %[1]U"
)

// statusColor converts an HTTP status code into a color ANSI string.
func statusColor(code int) string {
	var cc string
	switch {
	case code >= 200 && code < 300:
		cc = "37;42"
	case code >= 400 && code < 500:
		cc = "7;33;40"
	case code >= 500:
		cc = "33;41"
	default:
		cc = "30;47"
	}
	return fmt.Sprint("\x1b[1;", cc, "m ", code, " \x1b[0m")
}

/*
Format implements the fmt.Formatter interface, based on Apache HTTP's
CustomLog directive. This allows a Context object to have Sprintf verbs for
its values. See: https://httpd.apache.org/docs/2.4/mod/mod_log_config.html#formats

Verb	Description
----	---------------------------------------------------
%%  	Percent sign
%a  	Client remote address
%b  	Size of reponse in bytes, excluding headers. Or '-' if zero.
%#a 	Proxy client address, or unknown.
%h  	Remote hostname. Will perform lookup.
%l  	Remote ident, will write '-' (only for Apache log support).
%m  	Request method
%q  	Request query string.
%r  	Request line.
%#r 	Request line without protocol.
%s  	Response status code.
%#s 	Response status code and text.
%t  	Request time, as string.
%u  	Remote user, if any.
%v  	Request host name.
%A  	User agent.
%B  	Size of reponse in bytes, excluding headers.
%C  	Colorized status code. For console, using ANSI escape codes.
%D  	Time lapsed to serve request, in seconds.
%H  	Request protocol.
%I  	Bytes received.
%L  	Request ID.
%P  	Server port used.
%R  	Referer.
%U  	Request path.

*/
func (ctx *Context) Format(f fmt.State, c rune) {
	var str string

	p, pok := f.Precision()
	if !pok {
		p = -1
	}

	switch c {
	case 'a':
		if f.Flag('#') {
			str = ctx.ProxyClient()
		} else {
			str = ctx.Request.RemoteAddr
		}
	case 'b':
		if ctx.Bytes() == 0 {
			f.Write([]byte{45})
			return
		}
		fallthrough
	case 'B':
		str = strconv.Itoa(ctx.Bytes())
	case 'h':
		t := strings.Split(ctx.Request.RemoteAddr, ":")
		str = t[0]
	case 'l':
		f.Write([]byte{45})
		return
	case 'm':
		str = ctx.Request.Method
	case 'q':
		str = ctx.Request.URL.RawQuery
	case 'r':
		str = ctx.Request.Method + " " + ctx.Request.URL.RequestURI()
		if f.Flag('#') {
			break
		}
		str += " " + ctx.Request.Proto
	case 's':
		str = strconv.Itoa(ctx.Status())
		if f.Flag('#') {
			str += " " + http.StatusText(ctx.Status())
		}
	case 't':
		t := ctx.Info.GetTime("context.start_time")
		str = t.Format("[02/Jan/2006:15:04:05 -0700]")
	case 'u':
		// XXX: i dont think net/http sets User
		if ctx.Request.URL.User == nil {
			f.Write([]byte{45})
			return
		}
		str = ctx.Request.URL.User.Username()
	case 'v':
		str = ctx.Request.Host
	case 'A':
		str = ctx.Request.UserAgent()
	case 'C':
		str = statusColor(ctx.Status())
	case 'D':
		when := ctx.Info.GetTime("context.start_time")
		if when.IsZero() {
			f.Write([]byte("%!(BADTIME)"))
			return
		}
		pok = false
		str = strconv.FormatFloat(time.Since(when).Seconds(), 'f', p, 32)
	case 'H':
		str = ctx.Request.Proto
	case 'I':
		str = fmt.Sprintf("%d", ctx.Request.ContentLength)
	case 'L':
		str = ctx.Info.Get("context.request_id")
	case 'P':
		s := strings.Split(ctx.Request.Host, ":")
		if len(s) > 1 {
			str = s[1]
		} else {
			str = "80"
		}
	case 'R':
		str = ctx.Request.Referer()
	case 'U':
		str = ctx.Request.URL.Path
	}
	if pok {
		str = str[:p]
	}
	f.Write([]byte(str))
}

/*
FilterLog provides pre- and post-request event logs. It uses a custom
log format similar to the one used for Apache HTTP CustomLog directive.

	log := &FilterLog{}
	log.Println("FilterLog implements Logger.")

	// Context-specific format verbs
	log.Panicf("%C bad status", ctx)

*/
type FilterLog struct {
	// Logger is an interface that is based on Go's log package. Any logging
	// system that implements Logger can be used.
	// Defaults to the stdlog in 'log' package.
	Logger

	// PreLogFormat is the format for the pre-request log entry.
	// Leave empty if no log even is needed.
	// Default to empty (no pre-log)
	PreLogFormat string

	// PostLogFormat is the format for the post-request log entry.
	// Defaults to the value of LogFormatRelax
	PostLogFormat string
}

// Run processes the filter. No info is passed.
func (f *FilterLog) Run(next HandlerFunc) HandlerFunc {
	if f.Logger == nil {
		f.Logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	if f.PostLogFormat == "" {
		f.PostLogFormat = LogFormatRelax
	}

	return func(ctx *Context) {
		if f.PreLogFormat != "" {
			f.Printf(f.PreLogFormat, ctx)
		}

		next(ctx)

		f.Printf(f.PostLogFormat, ctx)
	}
}
