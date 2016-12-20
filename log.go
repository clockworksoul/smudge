package blackfish

import (
	"fmt"
	"os"
)

type LogLevel byte

const (
	LOG_ALL LogLevel = iota
	LOG_TRACE
	LOG_DEBUG
	LOG_INFO
	LOG_WARN
	LOG_ERROR
	LOG_FATAL
	LOG_OFF
)

var logThreshhold LogLevel = LOG_ALL

func (s LogLevel) String() string {
	switch s {
	case LOG_ALL:
		return "[ALL]"
	case LOG_TRACE:
		return "[TRACE]"
	case LOG_DEBUG:
		return "[DEBUG]"
	case LOG_INFO:
		return "[INFO]"
	case LOG_WARN:
		return "[WARN]"
	case LOG_ERROR:
		return "[ERROR]"
	case LOG_FATAL:
		return "[FATAL]"
	case LOG_OFF:
		return "[OFF]"
	default:
		return "[UNKNOWN]"
	}
}

func Log(level LogLevel, a ...interface{}) (n int, err error) {
	fmt.Fprint(os.Stdout, level.String()+" ")

	return fmt.Fprintln(os.Stdout, a...)
}

func LogTrace(a ...interface{}) (n int, err error) {
	return Log(LOG_TRACE, a...)
}

func LogDebug(a ...interface{}) (n int, err error) {
	return Log(LOG_DEBUG, a...)
}

func LogInfo(a ...interface{}) (n int, err error) {
	return Log(LOG_INFO, a...)
}

func LogWarn(a ...interface{}) (n int, err error) {
	return Log(LOG_WARN, a...)
}

func LogError(a ...interface{}) (n int, err error) {
	return Log(LOG_ERROR, a...)
}

func LogFatal(a ...interface{}) (n int, err error) {
	return Log(LOG_FATAL, a...)
}

func Logf(level LogLevel, format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(os.Stdout, level.String()+" "+format, a...)
}

func LogfTrace(format string, a ...interface{}) (n int, err error) {
	return Logf(LOG_TRACE, format, a...)
}

func LogfDebug(format string, a ...interface{}) (n int, err error) {
	return Logf(LOG_DEBUG, format, a...)
}

func LogfInfo(format string, a ...interface{}) (n int, err error) {
	return Logf(LOG_INFO, format, a...)
}

func LogfWarn(format string, a ...interface{}) (n int, err error) {
	return Logf(LOG_WARN, format, a...)
}

func LogfError(format string, a ...interface{}) (n int, err error) {
	return Logf(LOG_ERROR, format, a...)
}

func LogfFatal(format string, a ...interface{}) (n int, err error) {
	return Logf(LOG_FATAL, format, a...)
}

func SetLogThreshhold(level LogLevel) {
	logThreshhold = level
}
