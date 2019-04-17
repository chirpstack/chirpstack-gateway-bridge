package structs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringToEUI64(t *testing.T) {
	assert := require.New(t)

	tests := []struct {
		Value    string
		Expected EUI64
		Error    error
	}{
		{
			Value:    "::0",
			Expected: EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			Value:    "1::",
			Expected: EUI64{0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			Value:    "::a:b",
			Expected: EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x0b},
		},
		{
			Value:    "f::1",
			Expected: EUI64{0x00, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		},
		{
			Value:    "f:a123:f8:100",
			Expected: EUI64{0x00, 0x0f, 0xa1, 0x23, 0x00, 0xf8, 0x01, 0x00},
		},
		{
			Value:    "01-02-03-04-05-06-07-08",
			Expected: EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		},
	}

	for _, tst := range tests {
		var eui EUI64
		assert.Equal(tst.Error, eui.UnmarshalText([]byte(tst.Value)))
		assert.Equal(tst.Expected, eui)
	}
}
