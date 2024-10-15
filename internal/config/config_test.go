package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	type args struct {
		configPath string
	}

	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name: "correct yaml",
			args: args{
				configPath: "config_test.yml",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := ParseConfig(tt.args.configPath)

			require.NoError(t, gotErr)

			require.Equal(t, 3, len(got.VRF))

			require.True(t, got.HTTP.Enable)
			require.False(t, got.Pyroscope.Enable)

			require.Equal(t, uint64(1), got.TFController.BGPKeepAlive)
			require.Equal(t, uint64(3), got.TFController.BGPHoldTimer)

			require.Equal(t, uint64(30), got.VRF[0].BGPKeepAlive)
			require.Equal(t, uint64(90), got.VRF[0].BGPHoldTimer)
		})
	}
}
