package packets

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
)

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

// PullRespPayload represents the downstream JSON data structure.
type PullRespPayload struct {
	TXPK TXPK `json:"txpk"`
}

// TXPK contains a RF packet to be emitted and associated metadata.
type TXPK struct {
	Imme bool    `json:"imme"`           // Send packet immediately (will ignore tmst & time)
	RFCh uint8   `json:"rfch"`           // Concentrator "RF chain" used for TX (unsigned integer)
	Powe uint8   `json:"powe"`           // TX output power in dBm (unsigned integer, dBm precision)
	Ant  uint8   `json:"ant"`            // Antenna number on which signal has been received
	Brd  uint32  `json:"brd"`            // Concentrator board used for RX (unsigned integer)
	Tmst *uint32 `json:"tmst,omitempty"` // Send packet on a certain timestamp value (will ignore time)
	Tmms *int64  `json:"tmms,omitempty"` // Send packet at a certain GPS time (GPS synchronization required)
	Freq float64 `json:"freq"`           // TX central frequency in MHz (unsigned float, Hz precision)
	Modu string  `json:"modu"`           // Modulation identifier "LORA" or "FSK"
	DatR DatR    `json:"datr"`           // LoRa datarate identifier (eg. SF12BW500) || FSK datarate (unsigned, in bits per second)
	CodR string  `json:"codr,omitempty"` // LoRa ECC coding rate identifier
	FDev uint16  `json:"fdev,omitempty"` // FSK frequency deviation (unsigned integer, in Hz)
	NCRC bool    `json:"ncrc,omitempty"` // If true, disable the CRC of the physical layer (optional)
	IPol bool    `json:"ipol"`           // Lora modulation polarization inversion
	Prea uint16  `json:"prea,omitempty"` // RF preamble size (unsigned integer)
	Size uint16  `json:"size"`           // RF packet payload size in bytes (unsigned integer)
	Data []byte  `json:"data"`           // Base64 encoded RF packet payload, padding optional
}

// GetPullRespPacket returns a PullRespPacket for the given gw.DownlinkFrame.
func GetPullRespPacket(protoVersion uint8, randomToken uint16, frame gw.DownlinkFrame, index int) (PullRespPacket, error) {
	if index > len(frame.Items)-1 {
		return PullRespPacket{}, fmt.Errorf("invalid frame index: %d", index)
	}

	item := frame.Items[index]
	txInfo := item.GetTxInfo()

	packet := PullRespPacket{
		ProtocolVersion: protoVersion,
		RandomToken:     randomToken,
		Payload: PullRespPayload{
			TXPK: TXPK{
				Freq: float64(txInfo.GetFrequency()) / 1000000,
				Powe: uint8(txInfo.GetPower()),
				Modu: txInfo.GetModulation().String(),
				Size: uint16(len(item.PhyPayload)),
				Data: item.PhyPayload,
				Ant:  uint8(txInfo.GetAntenna()),
				Brd:  uint32(txInfo.GetBoard()),
			},
		},
	}

	if txInfo.GetModulation() == common.Modulation_LORA {
		modInfo := txInfo.GetLoraModulationInfo()
		if modInfo == nil {
			return packet, errors.New("gateway: lora_modulation_info must not be nil")
		}
		packet.Payload.TXPK.DatR.LoRa = fmt.Sprintf("SF%dBW%d", modInfo.SpreadingFactor, modInfo.Bandwidth)
		packet.Payload.TXPK.CodR = modInfo.CodeRate
		packet.Payload.TXPK.IPol = modInfo.PolarizationInversion
	}

	if txInfo.GetModulation() == common.Modulation_FSK {
		modInfo := txInfo.GetFskModulationInfo()
		if modInfo == nil {
			return packet, errors.New("gateway: fsk_modulation_info must not be nil")
		}
		packet.Payload.TXPK.DatR.FSK = modInfo.Datarate
		packet.Payload.TXPK.FDev = uint16(modInfo.FrequencyDeviation)

		// TODO: cleanup in next major release
		if packet.Payload.TXPK.FDev == 0 {
			packet.Payload.TXPK.FDev = uint16(modInfo.Datarate / 2)
		}
	}

	switch txInfo.GetTiming() {
	case gw.DownlinkTiming_IMMEDIATELY:
		packet.Payload.TXPK.Imme = true

	case gw.DownlinkTiming_DELAY:
		timingInfo := txInfo.GetDelayTimingInfo()
		if timingInfo == nil {
			return packet, errors.New("delay_timing_info must not be nil")
		}

		delay, err := ptypes.Duration(timingInfo.Delay)
		if err != nil {
			return packet, errors.Wrap(err, "get delay duration error")
		}

		if len(txInfo.GetContext()) < 4 {
			return packet, fmt.Errorf("context must contain at least 4 bytes, got: %d", len(txInfo.GetContext()))
		}
		timestamp := binary.BigEndian.Uint32(txInfo.GetContext()[0:4])
		timestamp += uint32(delay / time.Microsecond)
		packet.Payload.TXPK.Tmst = &timestamp

	case gw.DownlinkTiming_GPS_EPOCH:
		timingInfo := txInfo.GetGpsEpochTimingInfo()
		if timingInfo == nil {
			return packet, errors.New("gps_epoch_timing must not be nil")
		}

		dur, err := ptypes.Duration(timingInfo.TimeSinceGpsEpoch)
		if err != nil {
			return packet, errors.Wrap(err, "parse time_since_gps_epoch error")
		}

		durMS := int64(dur / time.Millisecond)
		packet.Payload.TXPK.Tmms = &durMS

	default:
		return packet, fmt.Errorf("unexpected downlink timing: %s", txInfo.GetTiming())
	}

	return packet, nil
}
