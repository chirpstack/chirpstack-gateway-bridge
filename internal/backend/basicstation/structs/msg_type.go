package structs

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// MessageType defines the message type.
type MessageType string

// Message types.
const (
	VersionMessage              MessageType = "version"
	RouterConfigMessage         MessageType = "router_config"
	JoinRequestMessage          MessageType = "jreq"
	UplinkDataFrameMessage      MessageType = "updf"
	ProprietaryDataFrameMessage MessageType = "propdf"
	DownlinkMessage             MessageType = "dnmsg"
	DownlinkTransmittedMessage  MessageType = "dntxed"
	TimeSyncMessage             MessageType = "timesync"
)

type messageTypePayload struct {
	MessageType MessageType `json:"msgtype"`
}

// GetMessageType returns the message type for the given paylaod.
func GetMessageType(b []byte) (MessageType, error) {
	var pl messageTypePayload
	if err := json.Unmarshal(b, &pl); err != nil {
		return "", errors.Wrap(err, "unmarshal message-type error")
	}

	return pl.MessageType, nil
}
