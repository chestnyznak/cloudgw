package netutils_test

import (
	"fmt"
	"net"
	"testing"

	"git.crptech.ru/cloud/cloudgw/pkg/netutils"

	"github.com/stretchr/testify/require"
)

func TestIsFIP(t *testing.T) {
	vppAggregatedFIPs := make([]*net.IPNet, 0)

	for _, fipPrefix := range []string{"10.0.0.0/24", "10.0.1.0/24", "192.168.0.0/24"} {
		_, parsedFIPPrefix, err := net.ParseCIDR(fipPrefix)
		require.NoError(t, err)

		vppAggregatedFIPs = append(vppAggregatedFIPs, parsedFIPPrefix)
	}

	tests := []struct {
		arg  string
		want bool
	}{
		{"10.0.0.1/32", true},
		{"10.0.1.1/32", true},
		{"10.0.1.33/32", true},
		{"10.0.2.1/32", false},
		{"192.168.0.1/24", true},
		{"192.168.1.1/24", false},
		{"", false},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test #%d", i), func(t *testing.T) {
			tt := tt
			require.Equal(t, tt.want, netutils.IsFIP(tt.arg, vppAggregatedFIPs))
		})
	}
}
