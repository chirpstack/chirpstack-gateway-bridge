package auth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		Name             string
		ConnectionString string
		ExpectedKV       map[string]string
		ExpectedError    error
	}{
		{
			Name:             "valid string",
			ConnectionString: "HostName=gateways-eu868.azure-devices.net;DeviceId=00800000a00016b6;SharedAccessKey=WWVQv+auegGaG2mm2/0FIS24xqkmZW/z5cYBO898+8I=",
			ExpectedKV: map[string]string{
				"HostName":        "gateways-eu868.azure-devices.net",
				"DeviceId":        "00800000a00016b6",
				"SharedAccessKey": "WWVQv+auegGaG2mm2/0FIS24xqkmZW/z5cYBO898+8I=",
			},
		},
		{
			Name:             "invalid string",
			ConnectionString: "HostName;gateways-eu868.azure-devices.net;DeviceId=00800000a00016b6;SharedAccessKey=WWVQv+auegGaG2mm2/0FIS24xqkmZW/z5cYBO898+8I=",
			ExpectedError:    errors.New("expected two items in: [HostName]"),
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)

			kv, err := parseConnectionString(tst.ConnectionString)
			assert.Equal(tst.ExpectedError, err)
			if err != nil {
				return
			}

			assert.EqualValues(tst.ExpectedKV, kv)
		})
	}
}
