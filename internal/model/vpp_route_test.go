package model_test

import (
	"testing"

	"git.crptech.ru/cloud/cloudgw/internal/model"

	"github.com/stretchr/testify/require"
	"go.fd.io/govpp/binapi/interface_types"
)

func TestAddPath(t *testing.T) {
	type args struct {
		nextHop   string
		tunnelID  uint32
		mplsLabel uint32
	}

	tests := []struct {
		name   string
		input  model.VPPIPRoute
		args   args
		output model.VPPIPRoute
	}{
		{
			"test 1",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
			args{
				"10.0.0.2",
				2,
				200,
			},
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1", "10.0.0.2"},
				TunnelIDs:       []uint32{1, 2},
				FIPMPLSLabels:   []uint32{100, 200},
			},
		},

		{
			"test 2",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
			args{
				"10.0.0.1",
				2,
				200,
			},
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{2},
				FIPMPLSLabels:   []uint32{200},
			},
		},

		{
			"test 3",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
			args{
				"",
				2,
				200,
			},
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
		},

		{
			"test 4",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
			args{
				"10.0.0.2",
				0,
				0,
			},
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt

			tt.input.AddPath(tt.args.nextHop, tt.args.tunnelID, tt.args.mplsLabel)

			require.Equal(t, tt.output, tt.input)
		})
	}
}

func TestDelPath(t *testing.T) {
	type args struct {
		nextHop string
	}

	tests := []struct {
		name   string
		input  model.VPPIPRoute
		args   args
		output model.VPPIPRoute
	}{
		{
			"test 1",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1", "10.0.0.2"},
				TunnelIDs:       []uint32{1, 2},
				FIPMPLSLabels:   []uint32{100, 200},
			},
			args{"10.0.0.2"},

			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
		},

		{
			"test 2",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1", "10.0.0.2"},
				TunnelIDs:       []uint32{1, 2},
				FIPMPLSLabels:   []uint32{100, 200},
			},
			args{"10.0.0.1"},

			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.2"},
				TunnelIDs:       []uint32{2},
				FIPMPLSLabels:   []uint32{200},
			},
		},

		{
			"test 3",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
				TunnelIDs:       []uint32{1, 2, 3},
				FIPMPLSLabels:   []uint32{100, 200, 300},
			},
			args{"10.0.0.2"},

			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1", "10.0.0.3"},
				TunnelIDs:       []uint32{1, 3},
				FIPMPLSLabels:   []uint32{100, 300},
			},
		},

		{
			"test 4",
			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        []string{"10.0.0.1"},
				TunnelIDs:       []uint32{1},
				FIPMPLSLabels:   []uint32{100},
			},
			args{"10.0.0.1"},

			model.VPPIPRoute{
				VRFID:           0,
				MainInterfaceID: interface_types.InterfaceIndex(1),
				SubInterfaceID:  interface_types.InterfaceIndex(2),
				Prefix:          "10.11.64.1/32",
				NextHops:        nil,
				TunnelIDs:       nil,
				FIPMPLSLabels:   nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt

			tt.input.DelPath(tt.args.nextHop)

			require.Equal(t, tt.output, tt.input)
		})
	}
}
