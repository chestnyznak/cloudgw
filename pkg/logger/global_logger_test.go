package logger

import (
	"testing"
)

func Test_InitGlobalLogger(_ *testing.T) {
	Init(
		WithLevel("info"),
		WithFormat("json"),
		WithOutput("stdout"),
	)

	Debug("debug test", "key", "value")
	Info("debug test", "key", "value")
	Warn("debug test", "key", "value")
	Error("debug test", "key", "value", "error", "not found")
}

func Test_initGlobalLogger(_ *testing.T) {
	Debug("debug test", "key", "value")
	Info("debug test", "key", "value")
	Warn("debug test", "key", "value")
	Error("debug test", "key", "value", "error", "not found")
}
