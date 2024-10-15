package netutils_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

func TestMPLSLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    uint32
		isError bool
	}{
		// expected = 10000000 + string(ipPrefix)[last 4 digits]
		{"1.2.3.4", 0, true},
		{"1.2.3.4/24/24", 0, true},
		{"1.2.3.1.24", 0, true},
		{"1.2.3.1\\24", 0, true},
		{"1.2.3./24", 0, true},
		{"1.2.3.4/24", 1001234, false},
		{"1.2.3.4/32", 1001234, false},
		{"0.0.0.0/0", 1000000, false},
		{"128.192.224.240/32", 1004240, false},
		{"10.11.12.13/24", 1001213, false},
		{"10.66.199.255/32", 1009255, false},
		{"192.0.1.2/29", 1002012, false},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			actual, err := netutils.MPLSLabel(tt.input)

			if tt.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, actual)
			require.True(t, actual < 1048575)
		})
	}
}
