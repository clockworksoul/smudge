/*
Copyright 2016 The Smudge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package smudge

import (
	"fmt"
	"os"
	"time"
)

// LogLevel represents a logging levels to be used as a parameter passed to
// the SetLogThreshhold() function.
type LogLevel byte

const (
	// LogAll allows all log output of all levels to be emitted.
	LogAll LogLevel = iota

	// LogTrace restricts log output to trace level and above.
	LogTrace

	// LogDebug restricts log output to debug level and above.
	LogDebug

	// LogInfo restricts log output to info level and above.
	LogInfo

	// LogWarn restricts log output to warn level and above.
	LogWarn

	// LogError restricts log output to error level and above.
	LogError

	// LogFatal restricts log output to fatal level.
	LogFatal

	// LogOff prevents all log output entirely.
	LogOff
)

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

// Logger should be implemented by Logger's that are passed via SetLogger.
type Logger interface {
	Log(level LogLevel, a ...interface{}) (int, error)
	Logf(level LogLevel, format string, a ...interface{}) (int, error)
}

// DefaultLogger is the default logger that is included with Smudge.
type DefaultLogger struct{}

var (
	logThreshhold LogLevel
	logger        Logger
)

// SetLogThreshold allows the output noise level to be adjusted by setting
// the logging priority threshold.
func SetLogThreshold(level LogLevel) {
	logThreshhold = level
}

// SetLogger plugs in another logger to control the output of the library
func SetLogger(l Logger) {
	logger = l
}

// Log writes a log message of a certain level to the logger
func (d DefaultLogger) Log(level LogLevel, a ...interface{}) (n int, err error) {
	if level >= logThreshhold {
		fmt.Fprint(os.Stderr, prefix(level)+" ")
		return fmt.Fprintln(os.Stderr, a...)
	}
	return 0, nil
}

// Logf writes a log message with a specific format to the logger
func (d DefaultLogger) Logf(level LogLevel, format string, a ...interface{}) (n int, err error) {
	if level >= logThreshhold {
		return fmt.Fprintf(os.Stderr, prefix(level)+" "+format+"\n", a...)
	}

	return 0, nil
}

func init() {
	SetLogger(DefaultLogger{})
	SetLogThreshold(LogInfo)
}

func log(level LogLevel, a ...interface{}) (n int, err error) {
	if level >= logThreshhold {
		return logger.Log(level, a...)
	}
	return 0, nil
}
func logf(level LogLevel, format string, a ...interface{}) (n int, err error) {
	if level >= logThreshhold {
		return logger.Logf(level, format, a...)
	}
	return 0, nil
}

func prefix(level LogLevel) string {
	f := time.Now().Format("02/Jan/2006:15:04:05 MST")

	return fmt.Sprintf("%5s %s -", level.String(), f)
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
