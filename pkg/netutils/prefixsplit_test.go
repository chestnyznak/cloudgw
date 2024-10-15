package netutils_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

func TestAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"1.2.3.4/24", "1.2.3.4"},
		{"0.0.0.0/24", "0.0.0.0"},
		{"0.0.0.0/0", "0.0.0.0"},
		{"1.2.3.4//24", ""},
		{"1.2.3.4/24/24", ""},
		{"1.2.3.1.24", ""},
		{"1.2.3.1\\24", ""},
		{"1.2.3./24", ""},
		{"1.2.3./32", ""},
		{"0/32", ""},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			actual := netutils.Addr(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}

func TestMaskLen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected uint32
	}{
		{"1.2.3.4/24", 24},
		{"0.0.0.0/24", 24},
		{"0.0.0.0/0", 0},
		{"1.2.3.4//24", 0},
		{"1.2.3.4/24/24", 0},
		{"1.2.3.1.24", 0},
		{"1.2.3.1\\24", 0},
		{"1.2.3./24", 0},
		{"1.2.3./32", 0},
		{"0/32", 0},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			actual := netutils.MaskLen(tt.input)
			require.Equal(t, tt.expected, actual)
		})
	}
}
