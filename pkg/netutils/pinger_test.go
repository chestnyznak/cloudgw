package netutils_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

func TestIsAddressAlive(t *testing.T) {
	t.Parallel()

	// localIP, err := getOwnIP("8.8.8.8")
	// if err != nil {
	// 	log.Print(err)
	//
	// 	localIP = "127.0.0.1"
	// }

	tests := []struct {
		inputAddress    string
		inputInterval   uint
		expectedOk      bool
		isErrorExpected bool
	}{
		// {
		// 	localIP,
		// 	1,
		// 	true,
		// 	false,
		// },
		{
			"127.0.0.1",
			1,
			true,
			false,
		},
		{
			"230.0.0.0",
			1,
			false,
			false,
		},
		{
			"0.0.0.0",
			1,
			false,
			false,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.inputAddress, func(t *testing.T) {
			t.Parallel()

			actualPingable, actualError := netutils.IsAddressAlive(tt.inputAddress, tt.inputInterval)

			if tt.expectedOk {
				require.True(t, actualPingable)
			} else {
				require.False(t, actualPingable)
			}

			if tt.isErrorExpected {
				require.Error(t, actualError)
			} else {
				require.NoError(t, actualError)
			}
		})
	}
}

// getOwnIP returns local IP address
// func getOwnIP(targetIP string) (string, error) {
// 	conn, err := net.Dial("udp", targetIP+":123")
// 	if err != nil {
// 		return "", err
// 	}
//
// 	defer func() {
// 		_ = conn.Close()
// 	}()
//
// 	localAddr := conn.LocalAddr().(*net.UDPAddr)
//
// 	return localAddr.IP.String(), nil
// }
