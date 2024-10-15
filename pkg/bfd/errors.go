package bfd

import "errors"

var (
	ErrBFDPacketDecode       = errors.New("bfd packet decode error")
	ErrNoBFDLayerFound       = errors.New("no bfd layer type found in packet")
	ErrBFDLayerMismatch      = errors.New("bfd layers mismatch: actual layers length less the wanted")
	ErrUnsupportedBFDVersion = errors.New("unsupported bfd protocol version")
	ErrBFDAuthHeader         = errors.New("bfd auth header error")
	ErrBFDAuthType           = errors.New("bfd auth type error")
	ErrBFDAuthKeyID          = errors.New("bfd auth key id error")
	ErrBFDAuthHeaderData     = errors.New("bfd auth header data error")
)
