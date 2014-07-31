package relax

import (
	"fmt"
	"log"
	"os"
)

// LogLevel indicates the severity of an event from LOG_EMERG (most imporant)
// to LOG_DEBUG (least important), use -1 to disable all logging.
// Events that are greater than the current Logger.SetLevel value are ignored.
// LogLevel values are based on Apache's LogLevel directive -
// https://httpd.apache.org/docs/2.4/mod/core.html#loglevel
type LogLevel int

const (
	LOG_EMERG  LogLevel = iota // emergency, system is unusable. Terminate.
	LOG_ALERT                  // action must be taken immediately
	LOG_CRIT                   // critical conditions
	LOG_ERR                    // error conditions
	LOG_WARN                   // warning conditions
	LOG_NOTICE                 // normal but significant condition
	LOG_INFO                   // informational
	LOG_DEBUG                  // terse, detailed debugging message.
)

// String returns a string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LOG_EMERG:
		return "EMERG"
	case LOG_ALERT:
		return "ALERT"
	case LOG_CRIT:
		return "CRIT"
	case LOG_ERR:
		return "ERROR"
	case LOG_WARN:
		return "WARN"
	case LOG_NOTICE:
		return "NOTICE"
	case LOG_INFO:
		return "INFO"
	case LOG_DEBUG:
		return "DEBUG"
	}
	return "???"
}

/*
Logger

The Logger interface allows any logging system to be plugged in.

Relax provides a very simple logging system that is intended to be replaced by
something more robust. The foundation is laid for logging systems that support
event levels. An object must implement the Logger interface to enhance logging.

Logging itself is individual to each application and it's almost impossible to
build a system that can handle all cases. Many Go packages implement competent
logging systems that should fit the Logger interface.

The default logging system is a slight enhancement of the log package with colored
prefixes for each event level.
*/
type Logger interface {
	// Print is analogous to log.Print.
	Print(LogLevel, ...interface{})

	// Printf is analogous to log.Printf.
	Printf(LogLevel, string, ...interface{})

	// Println is analogous to log.Println; adds spaces between values and appends a newline.
	Println(LogLevel, ...interface{})

	// SetLevel sets the minimum level value for a log event to be printed.
	SetLevel(LogLevel)
}

// Log is the global framework Logger.
var Log Logger

// logger implements the Logger interface. It's a simple log to os.Stderr with
// accent colors for the levels.
// log is a log.Logger object.
// level specifies the urgency of the log, events with level above this value are
// not logged.
type logger struct {
	log   *log.Logger
	level LogLevel
}

// LogLevel converts a log level int to a prefix string that represents it.
// ANSI escape sequences are used to colorize the prefix.
// See https://en.wikipedia.org/wiki/ANSI_escape_code#Colors
func (self *logger) LogLevel(level LogLevel) string {
	var format string
	switch level {
	case LOG_EMERG, LOG_ALERT, LOG_CRIT:
		format = "\x1b[1;33;41m!%c!\x1b[0m "
	case LOG_ERR:
		format = "\x1b[1;31m=%c=\x1b[0m "
	case LOG_WARN:
		format = "\x1b[1;33m=%c=\x1b[0m "
	case LOG_NOTICE:
		format = "\x1b[32m[%c]\x1b[0m "
	case LOG_DEBUG:
		format = "\x1b[34m[%c]\x1b[0m "
	default:
		format = "[%c] "
	}
	return fmt.Sprintf(format, level.String()[0])
}

func (self *logger) Print(level LogLevel, v ...interface{}) {
	if level > self.level {
		return
	}
	self.log.SetPrefix(self.LogLevel(level))
	self.log.Print(v...)
}

func (self *logger) Printf(level LogLevel, format string, v ...interface{}) {
	if level > self.level {
		return
	}
	self.log.SetPrefix(self.LogLevel(level))
	self.log.Printf(format, v...)
}

func (self *logger) Println(level LogLevel, v ...interface{}) {
	if level > self.level {
		return
	}
	self.log.SetPrefix(self.LogLevel(level))
	self.log.Println(v...)
}

func (self *logger) SetLevel(level LogLevel) {
	self.level = level
}

// StatusLogLevel converts an HTTP status code into a log level value.
// It returns one of the following levels:
// codes 100-199 = LOG_INFO, codes 200-299 = LOG_NOTICE, codes 400-499 = LOG_WARN,
// code 500+ = LOG_ERR
func StatusLogLevel(code int) LogLevel {
	level := LOG_INFO
	switch {
	case code >= 200 && code < 300:
		level = LOG_NOTICE
	case code >= 400 && code < 500:
		level = LOG_WARN
	case code >= 500:
		level = LOG_ERR
	}
	return level
}

// Logging allows a new logging system to be changed from the default one. logger
// is an object the implements Logger.
func Logging(logger Logger) {
	Log = logger
}

// DefaultLogger is a simple os.Stderr logger with levels and color. Each
// log message is prefixed with one of the following color-coded strings based
// on the event level. The initial log level is LOG_INFO.
// Log level prefixes:
// 	LOG_EMERG:  "!E!"
// 	LOG_ALERT:  "!A!"
// 	LOG_CRIT:   "!C!"
// 	LOG_ERR:    "=E="
// 	LOG_WARN:   "=W="
// 	LOG_NOTICE: "[N]"
// 	LOG_INFO:   "[I]"
// 	LOG_DEBUG:  "[D]"
var DefaultLogger = &logger{log.New(os.Stderr, "", log.LstdFlags), LOG_INFO}

func init() {
	Log = DefaultLogger
}
