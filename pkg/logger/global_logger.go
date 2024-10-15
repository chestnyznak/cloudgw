package logger

import (
	"log/slog"
	"os"
)

var globalLogger *slog.Logger

func init() {
	if globalLogger == nil {
		globalLogger = Default()
	}
}

func Init(opts ...func(*Config)) {
	globalLogger = New(opts...)
}

func Debug(msg string, args ...any) {
	globalLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	globalLogger.Error(msg, args...)
}

func Fatal(msg string, args ...any) {
	globalLogger.Error(msg, args...)
	os.Exit(1)
}

func GetLogger() *slog.Logger {
	return globalLogger
}
