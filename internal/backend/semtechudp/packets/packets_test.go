package packets

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDatR(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		DatR   DatR
		String string
	}{
		{
			DatR:   DatR{LoRa: "SF7BW125"},
			String: `"SF7BW125"`,
		},
		{
			DatR:   DatR{FSK: 50000},
			String: "50000",
		},
	}

	for _, test := range testTable {
		b, err := test.DatR.MarshalJSON()
		assert.Nil(err)
		assert.Equal(test.String, string(b))

		var datR DatR
		assert.Nil(datR.UnmarshalJSON([]byte(test.String)))
		assert.Equal(test.DatR, datR)
	}
}

func TestCompactTime(t *testing.T) {
	assert := assert.New(t)

	tStr := "Mon Jan 2 15:04:05 -0700 MST 2006"
	ts, err := time.Parse(tStr, tStr)
	assert.Nil(err)

	testTable := []struct {
		Time   CompactTime
		String string
	}{
		{
			Time:   CompactTime(ts),
			String: `"2006-01-02T22:04:05Z"`,
		},
		{
			Time:   CompactTime(time.Time{}),
			String: "null",
		},
	}

	for _, test := range testTable {
		b, err := test.Time.MarshalJSON()
		assert.Nil(err)
		assert.Equal(test.String, string(b))

		str := test.String
		if str == "null" {
			str = `""`
		}

		var cp CompactTime
		assert.Nil(cp.UnmarshalJSON([]byte(str)))
		assert.True(time.Time(test.Time).Equal(time.Time(cp)))
	}
}

func TestGetPacketType(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes      []byte
		PacketType PacketType
		Error      error
	}{
		{
			Bytes: []byte{},
			Error: errors.New("gateway: at least 4 bytes of data are expected"),
		},
		{
			Bytes: []byte{3, 1, 3, 4},
			Error: ErrInvalidProtocolVersion,
		},
		{
			Bytes:      []byte{2, 1, 3, 4},
			PacketType: PullACK,
		},
	}

	for _, test := range testTable {
		pt, err := GetPacketType(test.Bytes)
		assert.Equal(test.Error, err)
		assert.Equal(test.PacketType, pt)
	}
}
