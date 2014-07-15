package relax

import (
	"fmt"
	"log"
	"os"
)

// Logger interface allows any logging system to be plugged in.
// Log importance is described as int levels, 0 being the most important (LOG_EMERG)
// and 7 the least (LOG_DEBUG).
type Logger interface {
	// Print is analogous to log.Print.
	// It expects an int which is the log level.
	Print(int, ...interface{})

	// Printf is analogous to log.Printf.
	// It expects an int which is the log level and a format string.
	Printf(int, string, ...interface{})

	// Println is analogous to log.Println; adds spaces between values and appends a newline.
	// It expects an int which is the log level.
	Println(int, ...interface{})

	// SetLevel sets the minimum level value for a log event to be sent. No
	// value-checking is done. Set to -1 to disable all logging.
	// It expects a log level.
	SetLevel(int)
}

// Log is the global framework Logger.
var Log Logger

// Logging event levels. These indicate the severity of an event from LOG_EMERG
// (most imporant) to LOG_DEBUG (least important). Events that are greater than
// the current Logger.SetLevel value are ignored.
const (
	LOG_EMERG int = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARN
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

// logger implements the Logger interface. It's a simple log to os.Stderr with
// accent colors for the levels.
// log is a log.Logger object.
// level specifies the urgency of the log, events with level above this value are
// not logged.
type logger struct {
	log   *log.Logger
	level int
}

// loggerLevels is a map of level values to string prefixes that are used to highlight
// events in the log output. These work better with color.
var loggerLevels = map[int]string{
	LOG_EMERG:  "!E!",
	LOG_ALERT:  "!A!",
	LOG_CRIT:   "!C!",
	LOG_ERR:    "=E=",
	LOG_WARN:   "=W=",
	LOG_NOTICE: "[N]",
	LOG_INFO:   "[I]",
	LOG_DEBUG:  "[D]",
}

// LogLevel converts a log level int to a prefix string that represents it.
// ASCII escape sequences are used to colorize the prefix.
func (self *logger) LogLevel(level int) string {
	str, ok := loggerLevels[level]
	if !ok {
		str = loggerLevels[LOG_INFO]
	}
	format := "\x1b[%sm%-3s\x1b[0m "
	color := ""
	switch level {
	case LOG_EMERG, LOG_ALERT, LOG_CRIT:
		color = "1;33;41"
	case LOG_ERR:
		color = "1;31"
	case LOG_WARN:
		color = "1;33"
	case LOG_NOTICE:
		color = "36"
	case LOG_DEBUG:
		color = "34"
	// case LOG_INFO:
	// color = "32"
	default:
		color = "39"
	}
	return fmt.Sprintf(format, color, str)
}

func (self *logger) Print(level int, v ...interface{}) {
	if level > self.level {
		return
	}
	self.log.SetPrefix(self.LogLevel(level))
	self.log.Print(v...)
}

func (self *logger) Printf(level int, format string, v ...interface{}) {
	if level > self.level {
		return
	}
	self.log.SetPrefix(self.LogLevel(level))
	self.log.Printf(format, v...)
}

func (self *logger) Println(level int, v ...interface{}) {
	if level > self.level {
		return
	}
	self.log.SetPrefix(self.LogLevel(level))
	self.log.Println(v...)
}

func (self *logger) SetLevel(level int) {
	self.level = level
}

// StatusLogLevel converts an HTTP status code into a log level value.
// It returns one of the following levels:
// codes 100-199 = LOG_INFO, codes 200-299 = LOG_NOTICE, codes 400-499 = LOG_WARN,
// code 500+ = LOG_ERR
func StatusLogLevel(code int) int {
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

func newLogger() *logger {
	return &logger{log.New(os.Stderr, "", log.LstdFlags), LOG_INFO}
}

// DefaultLogger is a simple os.Stderr logger with levels and color. Each
// log message is prefixed with one of the following color-coded strings based
// on the event level. The initial log level is LOG_INFO.
// Log level prefixes:
// 	LOG_EMERG:  "!E!",
// 	LOG_ALERT:  "!A!",
// 	LOG_CRIT:   "!C!",
// 	LOG_ERR:    "=E=",
// 	LOG_WARN:   "=W=",
// 	LOG_NOTICE: "[N]",
// 	LOG_INFO:   "[I]",
// 	LOG_DEBUG:  "[D]",
var DefaultLogger = newLogger()

func init() {
	Log = DefaultLogger
}
