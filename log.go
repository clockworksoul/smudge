package blackfish

import (
	"fmt"
	"os"
)

type logLevel byte

const (
	LOG_ALL logLevel = iota
	LOG_TRACE
	LOG_DEBUG
	LOG_INFO
	LOG_WARN
	LOG_ERROR
	LOG_FATAL
	LOG_OFF
)

var logThreshhold logLevel = LOG_ALL

func (s logLevel) String() string {
	switch s {
	case LOG_ALL:
		return "ALL"
	case LOG_TRACE:
		return "TRACE"
	case LOG_DEBUG:
		return "DEBUG"
	case LOG_INFO:
		return "INFO"
	case LOG_WARN:
		return "WARN"
	case LOG_ERROR:
		return "ERROR"
	case LOG_FATAL:
		return "FATAL"
	case LOG_OFF:
		return "OFF"
	default:
		return "UNKNOWN"
	}
}

func SetLogThreshhold(level logLevel) {
	logThreshhold = level
}

func log(level logLevel, a ...interface{}) (n int, err error) {
	fmt.Fprint(os.Stdout, level.String()+" ")

	return fmt.Fprintln(os.Stdout, a...)
}

func logTrace(a ...interface{}) (n int, err error) {
	return log(LOG_TRACE, a...)
}

func logDebug(a ...interface{}) (n int, err error) {
	return log(LOG_DEBUG, a...)
}

func logInfo(a ...interface{}) (n int, err error) {
	return log(LOG_INFO, a...)
}

func logWarn(a ...interface{}) (n int, err error) {
	return log(LOG_WARN, a...)
}

func logError(a ...interface{}) (n int, err error) {
	return log(LOG_ERROR, a...)
}

func logFatal(a ...interface{}) (n int, err error) {
	return log(LOG_FATAL, a...)
}

func logf(level logLevel, format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(os.Stdout, level.String()+" "+format, a...)
}

func logfTrace(format string, a ...interface{}) (n int, err error) {
	return logf(LOG_TRACE, format, a...)
}

func logfDebug(format string, a ...interface{}) (n int, err error) {
	return logf(LOG_DEBUG, format, a...)
}

func logfInfo(format string, a ...interface{}) (n int, err error) {
	return logf(LOG_INFO, format, a...)
}

func logfWarn(format string, a ...interface{}) (n int, err error) {
	return logf(LOG_WARN, format, a...)
}

func logfError(format string, a ...interface{}) (n int, err error) {
	return logf(LOG_ERROR, format, a...)
}

func logfFatal(format string, a ...interface{}) (n int, err error) {
	return logf(LOG_FATAL, format, a...)
}
