package bfd

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

var auth = &layers.BFDAuthHeader{
	AuthType:       layers.BFDAuthTypeKeyedMD5,
	KeyID:          2,
	SequenceNumber: 5,
	Data:           []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16},
}

func DecodePacket(packetBytes []byte) (*layers.BFD, error) {
	var pBFD *layers.BFD

	p := gopacket.NewPacket(packetBytes, layers.LayerTypeBFD, gopacket.Default)

	if p.ErrorLayer() != nil {
		logger.Error("failed to decode bfd packet", "error", p.ErrorLayer().Error())

		return nil, ErrBFDPacketDecode
	}

	// packet contains "Application Layer = BFD"
	if err := checkLayers(p, []gopacket.LayerType{layers.LayerTypeBFD}); err != nil {
		return nil, err
	}

	pBFD, ok := p.ApplicationLayer().(*layers.BFD)
	if !ok { // no bfd protocol layer
		logger.Error("no bfd layer type found in packet")

		return nil, ErrNoBFDLayerFound
	}

	if err := validate(pBFD); err != nil {
		logger.Error("failed to validate bfd packet", "error", err)

		return nil, err
	}

	logger.Debug("bfd packet successfully decoded")

	return pBFD, nil
}

func checkLayers(p gopacket.Packet, want []gopacket.LayerType) error {
	packetLayers := p.Layers()

	if len(packetLayers) < len(want) {
		logger.Error("number of bfd packetLayers mismatch", "got", len(packetLayers), "wanted", len(want))

		return ErrBFDLayerMismatch
	}

	for i, l := range want {
		if l == gopacket.LayerTypePayload {
			// done matching packetLayers
			continue
		}

		if packetLayers[i].LayerType() != l {
			logger.Error(fmt.Sprintf("bfd layer %d mismatch: got %v want %v", i, packetLayers[i].LayerType(), l))

			return fmt.Errorf("bfd layer %d mismatch: got %v want %v", i, packetLayers[i].LayerType(), l)
		}
	}

	return nil
}

func validate(pBFD *layers.BFD) error {
	if pBFD.Version != 1 {
		return ErrUnsupportedBFDVersion
	}

	if pBFD.AuthPresent {
		if pBFD.AuthHeader == nil {
			return ErrBFDAuthHeader
		}

		if pBFD.AuthHeader.AuthType != auth.AuthType {
			return ErrBFDAuthType
		}

		if pBFD.AuthHeader.KeyID != auth.KeyID {
			return ErrBFDAuthKeyID
		}

		if string(pBFD.AuthHeader.Data) != string(auth.Data) {
			return ErrBFDAuthHeaderData
		}
	}

	return nil
}

// EncodePacket encodes a packet
func EncodePacket(
	version layers.BFDVersion,
	diagnostic layers.BFDDiagnostic,
	state layers.BFDState,
	poll bool,
	final bool,
	controlPlaneIndependent bool,
	authPresent bool,
	demand bool,
	multipoint bool,
	detectMultiplier layers.BFDDetectMultiplier,
	myDiscriminator layers.BFDDiscriminator,
	yourDiscriminator layers.BFDDiscriminator,
	desiredMinTxInterval layers.BFDTimeInterval,
	requiredMinRxInterval layers.BFDTimeInterval,
	requiredMinEchoRxInterval layers.BFDTimeInterval,
	authHeader *layers.BFDAuthHeader,
) []byte {
	pExpectedBFD := &layers.BFD{
		BaseLayer: layers.BaseLayer{
			Contents: []byte{
				0x20, 0x40, 0x05, 0x18, 0x00, 0x00, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x42, 0x40,
				0x00, 0x0f, 0x42, 0x40, 0x00, 0x00, 0x00, 0x00,
			},
			Payload: nil,
		},
		Version:                   version,
		Diagnostic:                diagnostic,
		State:                     state,
		Poll:                      poll,
		Final:                     final,
		ControlPlaneIndependent:   controlPlaneIndependent,
		AuthPresent:               authPresent,
		Demand:                    demand,
		Multipoint:                multipoint,
		DetectMultiplier:          detectMultiplier,
		MyDiscriminator:           myDiscriminator,
		YourDiscriminator:         yourDiscriminator,
		DesiredMinTxInterval:      desiredMinTxInterval,
		RequiredMinRxInterval:     requiredMinRxInterval,
		RequiredMinEchoRxInterval: requiredMinEchoRxInterval,
		AuthHeader:                authHeader,
	}

	buf := gopacket.NewSerializeBuffer()

	opts := gopacket.SerializeOptions{}

	if err := pExpectedBFD.SerializeTo(buf, opts); err != nil {
		logger.Error("failed to buffer serialization", "error", err)

		return []byte{}
	}

	return buf.Bytes()
}
