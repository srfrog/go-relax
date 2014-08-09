package relax

import (
	"fmt"
	"log"
	"os"
)

// LogLevel indicates the severity of an event from LogEmerg (most imporant)
// to LogDebug (least important), use -1 to disable all logging.
// Events that are greater than the current Logger.SetLevel value are ignored.
// LogLevel values are based on Apache's LogLevel directive -
// https://httpd.apache.org/docs/2.4/mod/core.html#loglevel
type LogLevel int

// Log level contants, 0 (LogEmerg) to 7 (LogDebug)
const (
	LogEmerg  LogLevel = iota // emergency, system is unusable. Terminate.
	LogAlert                  // action must be taken immediately
	LogCrit                   // critical conditions
	LogErr                    // error conditions
	LogWarn                   // warning conditions
	LogNotice                 // normal but significant condition
	LogInfo                   // informational
	LogDebug                  // terse, detailed debugging message.
)

// String returns a string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogEmerg:
		return "EMERG"
	case LogAlert:
		return "ALERT"
	case LogCrit:
		return "CRIT"
	case LogErr:
		return "ERROR"
	case LogWarn:
		return "WARN"
	case LogNotice:
		return "NOTICE"
	case LogInfo:
		return "INFO"
	case LogDebug:
		return "DEBUG"
	}
	return "???"
}

/*
Logger interface allows any logging system to be plugged in.

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
func (l *logger) LogLevel(level LogLevel) string {
	var format string
	switch level {
	case LogEmerg, LogAlert, LogCrit:
		format = "\x1b[1;33;41m!%c!\x1b[0m "
	case LogErr:
		format = "\x1b[1;31m=%c=\x1b[0m "
	case LogWarn:
		format = "\x1b[1;33m=%c=\x1b[0m "
	case LogNotice:
		format = "\x1b[32m[%c]\x1b[0m "
	case LogDebug:
		format = "\x1b[34m[%c]\x1b[0m "
	default:
		format = "[%c] "
	}
	return fmt.Sprintf(format, level.String()[0])
}

func (l *logger) Print(level LogLevel, v ...interface{}) {
	if level > l.level {
		return
	}
	l.log.SetPrefix(l.LogLevel(level))
	l.log.Print(v...)
}

func (l *logger) Printf(level LogLevel, format string, v ...interface{}) {
	if level > l.level {
		return
	}
	l.log.SetPrefix(l.LogLevel(level))
	l.log.Printf(format, v...)
}

func (l *logger) Println(level LogLevel, v ...interface{}) {
	if level > l.level {
		return
	}
	l.log.SetPrefix(l.LogLevel(level))
	l.log.Println(v...)
}

func (l *logger) SetLevel(level LogLevel) {
	l.level = level
}

// StatusLogLevel converts an HTTP status code into a log level value.
// It returns one of the following levels:
// codes 100-199 = LogInfo, codes 200-299 = LogNotice, codes 400-499 = LogWarn,
// code 500+ = LogErr
func StatusLogLevel(code int) LogLevel {
	level := LogInfo
	switch {
	case code >= 200 && code < 300:
		level = LogNotice
	case code >= 400 && code < 500:
		level = LogWarn
	case code >= 500:
		level = LogErr
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
// on the event level. The initial log level is LogInfo.
// Log level prefixes:
// 	LogEmerg:  "!E!"
// 	LogAlert:  "!A!"
// 	LogCrit:   "!C!"
// 	LogErr:    "=E="
// 	LogWarn:   "=W="
// 	LogNotice: "[N]"
// 	LogInfo:   "[I]"
// 	LogDebug:  "[D]"
var DefaultLogger = &logger{log.New(os.Stderr, "", log.LstdFlags), LogInfo}

func init() {
	Log = DefaultLogger
}
