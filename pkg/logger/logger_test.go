package logger

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	type args struct {
		level  string
		format string
		output string
		source bool
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "debug, json, stdout, no source",
			args: args{
				level:  "debug",
				format: "json",
				output: "stdout",
				source: false,
			},
		},
		{
			name: "info, json, stdout, source",
			args: args{
				level:  "info",
				format: "json",
				output: "stdout",
				source: true,
			},
		},
		{
			name: "warn, text, stdout, source",
			args: args{
				level:  "warn",
				format: "console",
				output: "stdout",
				source: true,
			},
		},
		{
			name: "error, text, stdout, no source",
			args: args{
				level:  "error",
				format: "console",
				output: "stdout",
				source: false,
			},
		},
		{
			name: "default (info), default (json), default (stdout), default (no source)",
			args: args{
				level:  "",
				format: "",
				output: "",
				source: false,
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := New(
				WithLevel(tt.args.level),
				WithFormat(tt.args.format),
				WithOutput(tt.args.output),
			)

			l.Debug(tt.name)
			l.Info(tt.name)
			l.Warn(tt.name)
			l.Error(tt.name)
		})
	}
}
