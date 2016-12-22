package blackfish

import (
	"fmt"
	"os"
	"time"
)

// The enumerated type representing logging levels, to be used as a parameter
// to the SetLogThreshhold() function.
type LogLevel byte

const (
	LogAll LogLevel = iota
	LogTrace
	LogDebug
	LogInfo
	LogWarn
	LogError
	LogFatal
	LogOff
)

var logThreshhold LogLevel = LogInfo

func (s LogLevel) String() string {
	switch s {
	case LogAll:
		return "All"
	case LogTrace:
		return "Trace"
	case LogDebug:
		return "Debug"
	case LogInfo:
		return "Info"
	case LogWarn:
		return "Warn"
	case LogError:
		return "Error"
	case LogFatal:
		return "Fatal"
	case LogOff:
		return "Off"
	default:
		return "Unknown"
	}
}

// Allows the output noise level to be adjusted by setting the logging threshhold.
func SetLogThreshhold(level LogLevel) {
	logThreshhold = level
}

func prefix(level LogLevel) string {
	f := time.Now().Format("02/Jan/2006:15:04:05 MST")

	return fmt.Sprintf("%5s %s -", level.String(), f)
}

func log(level LogLevel, a ...interface{}) (n int, err error) {
	if level >= logThreshhold {
		fmt.Fprint(os.Stdout, prefix(level)+" ")

		return fmt.Fprintln(os.Stdout, a...)
	}

	return 0, nil
}

func logTrace(a ...interface{}) (n int, err error) {
	return log(LogTrace, a...)
}

func logDebug(a ...interface{}) (n int, err error) {
	return log(LogDebug, a...)
}

func logInfo(a ...interface{}) (n int, err error) {
	return log(LogInfo, a...)
}

func logWarn(a ...interface{}) (n int, err error) {
	return log(LogWarn, a...)
}

func logError(a ...interface{}) (n int, err error) {
	return log(LogError, a...)
}

func logFatal(a ...interface{}) (n int, err error) {
	return log(LogFatal, a...)
}

func logf(level LogLevel, format string, a ...interface{}) (n int, err error) {
	if level >= logThreshhold {
		return fmt.Fprintf(os.Stdout, prefix(level)+" "+format, a...)
	}

	return 0, nil
}

func logfTrace(format string, a ...interface{}) (n int, err error) {
	return logf(LogTrace, format, a...)
}

func logfDebug(format string, a ...interface{}) (n int, err error) {
	return logf(LogDebug, format, a...)
}

func logfInfo(format string, a ...interface{}) (n int, err error) {
	return logf(LogInfo, format, a...)
}

func logfWarn(format string, a ...interface{}) (n int, err error) {
	return logf(LogWarn, format, a...)
}

func logfError(format string, a ...interface{}) (n int, err error) {
	return logf(LogError, format, a...)
}

func logfFatal(format string, a ...interface{}) (n int, err error) {
	return logf(LogFatal, format, a...)
}
