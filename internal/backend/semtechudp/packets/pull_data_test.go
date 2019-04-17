package packets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullDataTest(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes          []byte
		PullDataPacket PullDataPacket
	}{
		{
			Bytes:          []byte{2, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0},
			PullDataPacket: PullDataPacket{ProtocolVersion: ProtocolVersion2},
		},
		{
			Bytes: []byte{2, 123, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8},
			PullDataPacket: PullDataPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			},
		},
	}

	for _, test := range testTable {
		b, err := test.PullDataPacket.MarshalBinary()
		assert.Nil(err)
		assert.Equal(test.Bytes, b)

		var p PullDataPacket
		assert.Nil(p.UnmarshalBinary(test.Bytes))
		assert.Equal(test.PullDataPacket, p)
	}
}
