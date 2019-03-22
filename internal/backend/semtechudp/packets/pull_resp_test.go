package packets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullResp(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes          []byte
		PullRespPacket PullRespPacket
	}{
		{
			Bytes: []byte{2, 0, 0, 3, 123, 125},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
			},
		},
		{
			Bytes: []byte{2, 123, 0, 3, 123, 125},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			},
		},
	}

	for _, test := range testTable {
		b, err := test.PullRespPacket.MarshalBinary()
		assert.Nil(err)
		assert.Equal(test.Bytes[0:4], b[0:4])

		var p PullRespPacket
		assert.Nil(p.UnmarshalBinary(test.Bytes))
		assert.Equal(test.PullRespPacket, p)
	}
}
