//go:generate stringer -type=PacketType

package gateway

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/brocaar/lorawan"
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

func protocolSupported(p uint8) bool {
	if p == ProtocolVersion1 || p == ProtocolVersion2 {
		return true
	}
	return false
}

// PushDataPacket type is used by the gateway mainly to forward the RF packets
// received, and associated metadata, to the server.
type PushDataPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
	GatewayMAC      lorawan.EUI64
	Payload         PushDataPayload
}

// MarshalBinary marshals the object in binary form.
func (p PushDataPacket) MarshalBinary() ([]byte, error) {
	pb, err := json.Marshal(&p.Payload)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 4, len(pb)+12)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(PushData)
	out = append(out, p.GatewayMAC[0:len(p.GatewayMAC)]...)
	out = append(out, pb...)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PushDataPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 13 {
		return errors.New("gateway: at least 13 bytes are expected")
	}
	if data[3] != byte(PushData) {
		return errors.New("gateway: identifier mismatch (PUSH_DATA expected)")
	}

	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}

	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	for i := 0; i < 8; i++ {
		p.GatewayMAC[i] = data[4+i]
	}

	return json.Unmarshal(data[12:], &p.Payload)
}

// PushACKPacket is used by the server to acknowledge immediately all the
// PUSH_DATA packets received.
type PushACKPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
}

// MarshalBinary marshals the object in binary form.
func (p PushACKPacket) MarshalBinary() ([]byte, error) {
	out := make([]byte, 4)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(PushACK)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PushACKPacket) UnmarshalBinary(data []byte) error {
	if len(data) != 4 {
		return errors.New("gateway: 4 bytes of data are expected")
	}
	if data[3] != byte(PushACK) {
		return errors.New("gateway: identifier mismatch (PUSH_ACK expected)")
	}

	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	return nil
}

// PullDataPacket is used by the gateway to poll data from the server.
type PullDataPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
	GatewayMAC      [8]byte
}

// MarshalBinary marshals the object in binary form.
func (p PullDataPacket) MarshalBinary() ([]byte, error) {
	out := make([]byte, 4, 12)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(PullData)
	out = append(out, p.GatewayMAC[0:len(p.GatewayMAC)]...)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PullDataPacket) UnmarshalBinary(data []byte) error {
	if len(data) != 12 {
		return errors.New("gateway: 12 bytes of data are expected")
	}
	if data[3] != byte(PullData) {
		return errors.New("gateway: identifier mismatch (PULL_DATA expected)")
	}

	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	for i := 0; i < 8; i++ {
		p.GatewayMAC[i] = data[4+i]
	}
	return nil
}

// PullACKPacket is used by the server to confirm that the network route is
// open and that the server can send PULL_RESP packets at any time.
type PullACKPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
}

// MarshalBinary marshals the object in binary form.
func (p PullACKPacket) MarshalBinary() ([]byte, error) {
	out := make([]byte, 4)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(PullACK)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PullACKPacket) UnmarshalBinary(data []byte) error {
	if len(data) != 4 {
		return errors.New("gateway: 4 bytes of data are expected")
	}
	if data[3] != byte(PullACK) {
		return errors.New("gateway: identifier mismatch (PULL_ACK expected)")
	}
	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	return nil
}

// PullRespPacket is used by the server to send RF packets and associated
// metadata that will have to be emitted by the gateway.
type PullRespPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
	Payload         PullRespPayload
}

// MarshalBinary marshals the object in binary form.
func (p PullRespPacket) MarshalBinary() ([]byte, error) {
	pb, err := json.Marshal(&p.Payload)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 4, 4+len(pb))
	out[0] = p.ProtocolVersion

	if p.ProtocolVersion != ProtocolVersion1 {
		// these two bytes are unused in ProtocolVersion1
		binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	}
	out[3] = byte(PullResp)
	out = append(out, pb...)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *PullRespPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 5 {
		return errors.New("gateway: at least 5 bytes of data are expected")
	}
	if data[3] != byte(PullResp) {
		return errors.New("gateway: identifier mismatch (PULL_RESP expected)")
	}
	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	return json.Unmarshal(data[4:], &p.Payload)
}

// TXACKPacket is used by the gateway to send a feedback to the server
// to inform if a downlink request has been accepted or rejected by the
// gateway.
type TXACKPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
	GatewayMAC      lorawan.EUI64
	Payload         *TXACKPayload
}

// MarshalBinary marshals the object into binary form.
func (p TXACKPacket) MarshalBinary() ([]byte, error) {
	var pb []byte
	if p.Payload != nil {
		var err error
		pb, err = json.Marshal(p.Payload)
		if err != nil {
			return nil, err
		}
	}

	out := make([]byte, 4, len(pb)+12)
	out[0] = p.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], p.RandomToken)
	out[3] = byte(TXACK)
	out = append(out, p.GatewayMAC[:]...)
	out = append(out, pb...)
	return out, nil
}

// UnmarshalBinary decodes the object from binary form.
func (p *TXACKPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 12 {
		return errors.New("gateway: at least 12 bytes of data are expected")
	}
	if data[3] != byte(TXACK) {
		return errors.New("gateway: identifier mismatch (TXACK expected)")
	}
	if !protocolSupported(data[0]) {
		return ErrInvalidProtocolVersion
	}
	p.ProtocolVersion = data[0]
	p.RandomToken = binary.LittleEndian.Uint16(data[1:3])
	for i := 0; i < 8; i++ {
		p.GatewayMAC[i] = data[4+i]
	}
	if len(data) > 12 {
		p.Payload = &TXACKPayload{}
		return json.Unmarshal(data[12:], p.Payload)
	}
	return nil
}

// TXACKPayload contains the TXACKPacket payload.
type TXACKPayload struct {
	TXPKACK TXPKACK `json:"txpk_ack"`
}

// TXPKACK contains the status information of the associated PULL_RESP
// packet.
type TXPKACK struct {
	Error string `json:"error"`
}

// PushDataPayload represents the upstream JSON data structure.
type PushDataPayload struct {
	RXPK []RXPK `json:"rxpk,omitempty"`
	Stat *Stat  `json:"stat,omitempty"`
}

// PullRespPayload represents the downstream JSON data structure.
type PullRespPayload struct {
	TXPK TXPK `json:"txpk"`
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

// DatR implements the data rate which can be either a string (LoRa identifier)
// or an unsigned integer in case of FSK (bits per second).
type DatR struct {
	LoRa string
	FSK  uint32
}

// MarshalJSON implements the json.Marshaler interface.
func (d DatR) MarshalJSON() ([]byte, error) {
	if d.LoRa != "" {
		return []byte(`"` + d.LoRa + `"`), nil
	}
	return []byte(strconv.FormatUint(uint64(d.FSK), 10)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *DatR) UnmarshalJSON(data []byte) error {
	i, err := strconv.ParseUint(string(data), 10, 32)
	if err != nil {
		d.LoRa = strings.Trim(string(data), `"`)
		return nil
	}
	d.FSK = uint32(i)
	return nil
}

// RXPK contain a RF packet and associated metadata.
type RXPK struct {
	Time *CompactTime `json:"time"` // UTC time of pkt RX, us precision, ISO 8601 'compact' format (e.g. 2013-03-31T16:21:17.528002Z)
	Tmms *int64       `json:"tmms"` // GPS time of pkt RX, number of milliseconds since 06.Jan.1980
	Tmst uint32       `json:"tmst"` // Internal timestamp of "RX finished" event (32b unsigned)
	Freq float64      `json:"freq"` // RX central frequency in MHz (unsigned float, Hz precision)
	Brd  uint8        `json:"brd"`  // Concentrator board used for RX (unsigned integer)
	Chan uint8        `json:"chan"` // Concentrator "IF" channel used for RX (unsigned integer)
	RFCh uint8        `json:"rfch"` // Concentrator "RF chain" used for RX (unsigned integer)
	Stat int8         `json:"stat"` // CRC status: 1 = OK, -1 = fail, 0 = no CRC
	Modu string       `json:"modu"` // Modulation identifier "LORA" or "FSK"
	DatR DatR         `json:"datr"` // LoRa datarate identifier (eg. SF12BW500) || FSK datarate (unsigned, in bits per second)
	CodR string       `json:"codr"` // LoRa ECC coding rate identifier
	RSSI int16        `json:"rssi"` // RSSI in dBm (signed integer, 1 dB precision)
	LSNR float64      `json:"lsnr"` // Lora SNR ratio in dB (signed float, 0.1 dB precision)
	Size uint16       `json:"size"` // RF packet payload size in bytes (unsigned integer)
	Data string       `json:"data"` // Base64 encoded RF packet payload, padded
	RSig []RSig       `json:"rsig"` // Received signal information, per antenna (Optional)
}

// RSig contains the received signal information per antenna.
type RSig struct {
	Ant   uint8   `json:"ant"`   // Antenna number on which signal has been received
	Chan  uint8   `json:"chan"`  // Concentrator "IF" channel used for RX (unsigned integer)
	LSNR  float64 `json:"lsnr"`  // Lora SNR ratio in dB (signed float, 0.1 dB precision)
	RSSIC int16   `json:"rssic"` // RSSI in dBm of the channel (signed integer, 1 dB precision)
}

// Stat contains the status of the gateway.
type Stat struct {
	Time ExpandedTime `json:"time"` // UTC 'system' time of the gateway, ISO 8601 'expanded' format (e.g 2014-01-12 08:59:28 GMT)
	Lati *float64     `json:"lati"` // GPS latitude of the gateway in degree (float, N is +)
	Long *float64     `json:"long"` // GPS latitude of the gateway in degree (float, E is +)
	Alti *int32       `json:"alti"` // GPS altitude of the gateway in meter RX (integer)
	RXNb uint32       `json:"rxnb"` // Number of radio packets received (unsigned integer)
	RXOK uint32       `json:"rxok"` // Number of radio packets received with a valid PHY CRC
	RXFW uint32       `json:"rxfw"` // Number of radio packets forwarded (unsigned integer)
	ACKR float64      `json:"ackr"` // Percentage of upstream datagrams that were acknowledged
	DWNb uint32       `json:"dwnb"` // Number of downlink datagrams received (unsigned integer)
	TXNb uint32       `json:"txnb"` // Number of packets emitted (unsigned integer)
}

// TXPK contains a RF packet to be emitted and associated metadata.
type TXPK struct {
	Imme bool    `json:"imme"`           // Send packet immediately (will ignore tmst & time)
	Tmst *uint32 `json:"tmst,omitempty"` // Send packet on a certain timestamp value (will ignore time)
	Tmms *int64  `json:"tmms,omitempty"` // Send packet at a certain GPS time (GPS synchronization required)
	Freq float64 `json:"freq"`           // TX central frequency in MHz (unsigned float, Hz precision)
	RFCh uint8   `json:"rfch"`           // Concentrator "RF chain" used for TX (unsigned integer)
	Powe uint8   `json:"powe"`           // TX output power in dBm (unsigned integer, dBm precision)
	Modu string  `json:"modu"`           // Modulation identifier "LORA" or "FSK"
	DatR DatR    `json:"datr"`           // LoRa datarate identifier (eg. SF12BW500) || FSK datarate (unsigned, in bits per second)
	CodR string  `json:"codr,omitempty"` // LoRa ECC coding rate identifier
	FDev uint16  `json:"fdev,omitempty"` // FSK frequency deviation (unsigned integer, in Hz)
	IPol bool    `json:"ipol"`           // Lora modulation polarization inversion
	Prea uint16  `json:"prea,omitempty"` // RF preamble size (unsigned integer)
	Size uint16  `json:"size"`           // RF packet payload size in bytes (unsigned integer)
	NCRC bool    `json:"ncrc,omitempty"` // If true, disable the CRC of the physical layer (optional)
	Data string  `json:"data"`           // Base64 encoded RF packet payload, padding optional
	Brd  uint8   `json:"brd"`            // Concentrator board used for RX (unsigned integer)
	Ant  uint8   `json:"ant"`            // Antenna number on which signal has been received
}

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
