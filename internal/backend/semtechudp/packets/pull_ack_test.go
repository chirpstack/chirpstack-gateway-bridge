package packets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullACK(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes         []byte
		PullACKPacket PullACKPacket
	}{
		{
			Bytes:         []byte{2, 0, 0, 4},
			PullACKPacket: PullACKPacket{ProtocolVersion: ProtocolVersion2},
		},
		{
			Bytes: []byte{2, 123, 0, 4},
			PullACKPacket: PullACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			},
		},
	}

	for _, test := range testTable {
		b, err := test.PullACKPacket.MarshalBinary()
		assert.Nil(err)
		assert.Equal(test.Bytes, b)

		var p PullACKPacket
		assert.Nil(p.UnmarshalBinary(test.Bytes))
		assert.Equal(test.PullACKPacket, p)
	}
}
