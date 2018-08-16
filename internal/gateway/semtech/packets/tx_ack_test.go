package packets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTXACK(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes       []byte
		TXACKPacket TXACKPacket
	}{
		{
			Bytes:       []byte{2, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0},
			TXACKPacket: TXACKPacket{ProtocolVersion: ProtocolVersion2},
		},
		{
			Bytes: []byte{2, 123, 0, 5, 8, 7, 6, 5, 4, 3, 2, 1},
			TXACKPacket: TXACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				GatewayMAC:      [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
			},
		},
		{
			Bytes: []byte{2, 123, 0, 5, 8, 7, 6, 5, 4, 3, 2, 1, 0},
			TXACKPacket: TXACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				GatewayMAC:      [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
			},
		},
		{
			Bytes: []byte{2, 123, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 123, 34, 116, 120, 112, 107, 95, 97, 99, 107, 34, 58, 123, 34, 101, 114, 114, 111, 114, 34, 58, 34, 67, 79, 76, 76, 73, 83, 73, 79, 78, 95, 66, 69, 65, 67, 79, 78, 34, 125, 125},
			TXACKPacket: TXACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				Payload: &TXACKPayload{
					TXPKACK: TXPKACK{
						Error: "COLLISION_BEACON",
					},
				},
			},
		},
	}

	for _, test := range testTable {
		b, err := test.TXACKPacket.MarshalBinary()
		assert.Nil(err)

		// iBTS 0 byte when no payload
		if len(test.Bytes) == 13 {
			assert.Equal(test.Bytes[:12], b)
		} else {
			assert.Equal(test.Bytes, b)
		}

		var p TXACKPacket
		assert.Nil(p.UnmarshalBinary(test.Bytes))
		assert.Equal(test.TXACKPacket, p)
	}
}
