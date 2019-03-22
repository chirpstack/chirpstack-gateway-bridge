package packets

import (
	"encoding/binary"
	"errors"
)

// PullDataPacket is used by the gateway to poll data from the server.
type PullDataPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
	GatewayMAC      [8]byte
}

// MarshalBinary marshals the object in binary form.
func (p PullDataPacket) MarshalBinary() ([]byte, error) {
	out := make([]byte, 4, 12)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(PullData)
	out = append(out, p.GatewayMAC[0:len(p.GatewayMAC)]...)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PullDataPacket) UnmarshalBinary(data []byte) error {
	if len(data) != 12 {
		return errors.New("gateway: 12 bytes of data are expected")
	}
	if data[3] != byte(PullData) {
		return errors.New("gateway: identifier mismatch (PULL_DATA expected)")
	}

	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	for i := 0; i < 8; i++ {
		p.GatewayMAC[i] = data[4+i]
	}
	return nil
}
