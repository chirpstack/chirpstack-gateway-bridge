package packets

import (
	"encoding/binary"
	"errors"
)

// PullACKPacket is used by the server to confirm that the network route is
// open and that the server can send PULL_RESP packets at any time.
type PullACKPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
}

// MarshalBinary marshals the object in binary form.
func (p PullACKPacket) MarshalBinary() ([]byte, error) {
	out := make([]byte, 4)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(PullACK)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PullACKPacket) UnmarshalBinary(data []byte) error {
	if len(data) != 4 {
		return errors.New("gateway: 4 bytes of data are expected")
	}
	if data[3] != byte(PullACK) {
		return errors.New("gateway: identifier mismatch (PULL_ACK expected)")
	}
	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	return nil
}
