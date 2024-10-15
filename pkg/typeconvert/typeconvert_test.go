package typeconvert_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"git.crptech.ru/cloud/cloudgw/pkg/typeconvert"
)

func TestUint32ToStr(t *testing.T) {
	tests := []struct {
		input          uint32
		expectedOutput string
	}{
		{
			10,
			"10",
		},
		{
			0,
			"0",
		},
		{
			0o0001,
			"1",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
			actualOutput := typeconvert.Uint32ToStr(tc.input)
			require.Equal(t, tc.expectedOutput, actualOutput)
		})
	}
}

func TestStrToUint32(t *testing.T) {
	tests := []struct {
		input           string
		expectedOutput  uint32
		isErrorExpected bool
	}{
		{
			"10",
			10,
			false,
		},
		{
			"123",
			123,
			false,
		},
		{
			"0",
			0,
			false,
		},
		{
			"1",
			1,
			false,
		},
		{
			"-1",
			0,
			true,
		},
		{
			"1a",
			0,
			true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("%v", tc.input), func(t *testing.T) {
			actualOutput, actualErr := typeconvert.StrToUint32(tc.input)

			require.Equal(t, tc.expectedOutput, actualOutput)

			if tc.isErrorExpected {
				require.Error(t, actualErr)
			} else {
				require.NoError(t, actualErr)
			}
		})
	}
}
