package logger

import (
	"io"
	"log/slog"
	"os"
)

type Config struct {
	level  string
	format string
	output string
}

func New(opts ...func(*Config)) *slog.Logger {
	cfg := &Config{
		level:  "info",
		format: "json",
		output: "stdout",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	level := logLevel(cfg.level)
	output := logOutput(cfg.output)

	switch cfg.format {
	case "console":
		opts := slog.HandlerOptions{Level: level}

		handler := slog.NewTextHandler(output, &opts)

		return slog.New(handler)

	default:
		opts := slog.HandlerOptions{Level: level}

		handler := slog.NewJSONHandler(output, &opts)

		return slog.New(handler)
	}
}

func logLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func logOutput(output string) io.Writer {
	switch output {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		return os.Stdout
	}
}

func Default() *slog.Logger {
	return New()
}

func WithLevel(level string) func(*Config) {
	return func(s *Config) {
		s.level = level
	}
}

func WithFormat(format string) func(*Config) {
	return func(s *Config) {
		s.format = format
	}
}

func WithOutput(output string) func(*Config) {
	return func(s *Config) {
		s.output = output
	}
}
