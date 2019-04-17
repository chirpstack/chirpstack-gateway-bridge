package packets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushACKPacket(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes         []byte
		PushACKPacket PushACKPacket
	}{
		{
			Bytes:         []byte{2, 0, 0, 1},
			PushACKPacket: PushACKPacket{ProtocolVersion: ProtocolVersion2},
		},
		{
			Bytes: []byte{2, 123, 0, 1},
			PushACKPacket: PushACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			},
		},
	}

	for _, test := range testTable {
		b, err := test.PushACKPacket.MarshalBinary()
		assert.Nil(err)
		assert.Equal(test.Bytes, b)

		var pap PushACKPacket
		assert.Nil(pap.UnmarshalBinary(test.Bytes))
		assert.Equal(test.PushACKPacket, pap)
	}
}
