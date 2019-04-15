package structs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMessageType(t *testing.T) {
	assert := require.New(t)

	jsonStr := `{"msgtype": "updf"}`

	typ, err := GetMessageType([]byte(jsonStr))
	assert.NoError(err)
	assert.Equal(UplinkDataFrameMessage, typ)

}
