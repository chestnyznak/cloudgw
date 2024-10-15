package gobgplogger

import (
	"os"
	"strings"

	bgplog "github.com/osrg/gobgp/v3/pkg/log"
	"github.com/sirupsen/logrus"
)

type LogLevel uint32

const (
	PanicLevel LogLevel = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

type Fields map[string]interface{}

type Logger interface {
	Panic(msg string, fields Fields)
	Fatal(msg string, fields Fields)
	Error(msg string, fields Fields)
	Warn(msg string, fields Fields)
	Info(msg string, fields Fields)
	Debug(msg string, fields Fields)
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

type GoBGPLogger struct {
	logger *logrus.Logger
}

func (l *GoBGPLogger) Panic(msg string, fields bgplog.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Panic(msg)
}

func (l *GoBGPLogger) Fatal(msg string, fields bgplog.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Fatal(msg)
}

func (l *GoBGPLogger) Error(msg string, fields bgplog.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Error(msg)
}

func (l *GoBGPLogger) Warn(msg string, fields bgplog.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Warn(msg)
}

func (l *GoBGPLogger) Info(msg string, fields bgplog.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Info(msg)
}

func (l *GoBGPLogger) Debug(msg string, fields bgplog.Fields) {
	l.logger.WithFields(logrus.Fields(fields)).Debug(msg)
}

func (l *GoBGPLogger) SetLevel(level bgplog.LogLevel) {
	l.logger.SetLevel(logrus.Level(level))
}

func (l *GoBGPLogger) GetLevel() bgplog.LogLevel {
	return bgplog.LogLevel(l.logger.GetLevel())
}

func NewGoBGPLogger(logLevel, logFormat, logOutput string) *GoBGPLogger {
	logger := logrus.New()

	switch strings.ToLower(logLevel) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	switch strings.ToLower(logOutput) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		logger.SetOutput(os.Stdout)
	}

	if strings.ToLower(logFormat) == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})

		return &GoBGPLogger{logger: logger}
	}

	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:            false,
		DisableColors:          true,
		DisableTimestamp:       false,
		FullTimestamp:          true,
		DisableLevelTruncation: false,
		PadLevelText:           false,
	})

	return &GoBGPLogger{logger: logger}
}
