//go:generate stringer -type=PacketType

package packets

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// PacketType defines the packet type.
type PacketType byte

// Available packet types
const (
	PushData PacketType = iota
	PushACK
	PullData
	PullResp
	PullACK
	TXACK
)

// Protocol versions
const (
	ProtocolVersion1 uint8 = 0x01
	ProtocolVersion2 uint8 = 0x02
)

// Errors
var (
	ErrInvalidProtocolVersion = errors.New("gateway: invalid protocol version")
)

// GetPacketType returns the packet type for the given packet data.
func GetPacketType(data []byte) (PacketType, error) {
	if len(data) < 4 {
		return PacketType(0), errors.New("gateway: at least 4 bytes of data are expected")
	}

	if !protocolSupported(data[0]) {
		return PacketType(0), ErrInvalidProtocolVersion
	}

	return PacketType(data[3]), nil
}

func protocolSupported(p uint8) bool {
	if p == ProtocolVersion1 || p == ProtocolVersion2 {
		return true
	}
	return false
}

// ExpandedTime implements time.Time but (un)marshals to and from
// ISO 8601 'expanded' format.
type ExpandedTime time.Time

// MarshalJSON implements the json.Marshaler interface.
func (t ExpandedTime) MarshalJSON() ([]byte, error) {
	return []byte(time.Time(t).UTC().Format(`"2006-01-02 15:04:05 MST"`)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *ExpandedTime) UnmarshalJSON(data []byte) error {
	t2, err := time.Parse(`"2006-01-02 15:04:05 MST"`, string(data))
	if err != nil {
		return err
	}
	*t = ExpandedTime(t2)
	return nil
}

// CompactTime implements time.Time but (un)marshals to and from
// ISO 8601 'compact' format.
type CompactTime time.Time

// MarshalJSON implements the json.Marshaler interface.
func (t CompactTime) MarshalJSON() ([]byte, error) {
	t2 := time.Time(t)
	if t2.IsZero() {
		return []byte("null"), nil
	}
	return []byte(t2.UTC().Format(`"` + time.RFC3339Nano + `"`)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *CompactTime) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		return nil
	}

	t2, err := time.Parse(`"`+time.RFC3339Nano+`"`, string(data))
	if err != nil {
		return err
	}
	*t = CompactTime(t2)
	return nil
}

// DatR implements the data rate which can be either a string (LoRa identifier)
// or an unsigned integer in case of FSK (bits per second).
type DatR struct {
	LRFHSS string
	LoRa   string
	FSK    uint32
}

// MarshalJSON implements the json.Marshaler interface.
func (d DatR) MarshalJSON() ([]byte, error) {
	if d.LoRa != "" {
		return []byte(`"` + d.LoRa + `"`), nil
	}
	if d.LRFHSS != "" {
		return []byte(`"` + d.LRFHSS + `"`), nil
	}
	return []byte(strconv.FormatUint(uint64(d.FSK), 10)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *DatR) UnmarshalJSON(data []byte) error {
	i, err := strconv.ParseUint(string(data), 10, 32)
	if err != nil {
		// remove the trailing and leading quotes
		str := strings.Trim(string(data), `"`)

		if strings.HasPrefix(str, "SF") {
			d.LoRa = str
		} else {
			d.LRFHSS = str
		}
		return nil
	}
	d.FSK = uint32(i)
	return nil
}
