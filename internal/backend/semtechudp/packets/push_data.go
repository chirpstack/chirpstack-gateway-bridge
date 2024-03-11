package packets

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/brocaar/lorawan"
	"github.com/chirpstack/chirpstack/api/go/v4/common"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

// loRaDataRateRegex contains a regexp for parsing the LoRa data-rate string.
var loRaDataRateRegex = regexp.MustCompile(`SF(\d+)BW(\d+)`)

// lrFHSSDataRateRegex contains the regexp for parsing the LR-FHSS data-rate string.
var lrFHSSDataRateRegex = regexp.MustCompile(`M0CW(\d+)`)

// PushDataPacket type is used by the gateway mainly to forward the RF packets
// received, and associated metadata, to the server.
type PushDataPacket struct {
	ProtocolVersion uint8
	RandomToken     uint16
	GatewayMAC      lorawan.EUI64
	Payload         PushDataPayload
}

// MarshalBinary encodes the packet into binary form compatible with the
// Semtech UDP protocol.
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

// GetGatewayStats returns the gw.GatewayStats object (if the packet contains stats).
func (p PushDataPacket) GetGatewayStats() (*gw.GatewayStats, error) {
	if p.Payload.Stat == nil {
		return nil, nil
	}

	stats := gw.GatewayStats{
		GatewayId:           p.GatewayMAC.String(),
		RxPacketsReceived:   p.Payload.Stat.RXNb,
		RxPacketsReceivedOk: p.Payload.Stat.RXOK,
		TxPacketsEmitted:    p.Payload.Stat.TXNb,
		TxPacketsReceived:   p.Payload.Stat.DWNb,
		Metadata:            p.Payload.Stat.Meta,
	}

	// time
	stats.Time = timestamppb.New(time.Time(p.Payload.Stat.Time))

	// location
	if p.Payload.Stat.Lati != 0 || p.Payload.Stat.Long != 0 || p.Payload.Stat.Alti != 0 {
		stats.Location = &common.Location{
			Latitude:  p.Payload.Stat.Lati,
			Longitude: p.Payload.Stat.Long,
			Altitude:  float64(p.Payload.Stat.Alti),
			Source:    common.LocationSource_GPS,
		}
	}

	return &stats, nil
}

// GetUplinkFrames returns a slice of gw.UplinkFrame.
func (p PushDataPacket) GetUplinkFrames(skipCRCCheck bool, FakeRxInfoTime bool) ([]*gw.UplinkFrame, error) {
	var frames []*gw.UplinkFrame

	for i := range p.Payload.RXPK {
		// validate CRC
		if p.Payload.RXPK[i].Stat != 1 && !skipCRCCheck {
			continue
		}

		if len(p.Payload.RXPK[i].RSig) == 0 {
			frame, err := getUplinkFrame(p.GatewayMAC, p.Payload.Stat, p.Payload.RXPK[i], FakeRxInfoTime)
			if err != nil {
				return nil, errors.Wrap(err, "backend/semtechudp/packets: get uplink frame error")
			}
			frame.RxInfo.UplinkId = uint32(p.RandomToken)
			frames = append(frames, frame)
		} else {
			for j := range p.Payload.RXPK[i].RSig {
				frame, err := getUplinkFrame(p.GatewayMAC, p.Payload.Stat, p.Payload.RXPK[i], FakeRxInfoTime)
				if err != nil {
					return nil, errors.Wrap(err, "backend/semtechudp/packets: get uplink frame error")
				}
				frame = setUplinkFrameRSig(frame, p.Payload.RXPK[i], p.Payload.RXPK[i].RSig[j])
				frame.RxInfo.UplinkId = uint32(p.RandomToken)

				frames = append(frames, frame)
			}
		}
	}

	return frames, nil
}

func setUplinkFrameRSig(frame *gw.UplinkFrame, rxPK RXPK, rSig RSig) *gw.UplinkFrame {
	frame.RxInfo.Antenna = uint32(rSig.Ant)
	frame.RxInfo.Channel = uint32(rSig.Chan)
	frame.RxInfo.Rssi = int32(rSig.RSSIC)
	frame.RxInfo.Snr = rSig.LSNR

	if len(rSig.ETime) != 0 {
		// TODO: handle decrypt and expose as plain fine-timestamp.
	}

	return frame
}

func getUplinkFrame(gatewayID lorawan.EUI64, stat *Stat, rxpk RXPK, FakeRxInfoTime bool) (*gw.UplinkFrame, error) {
	frame := gw.UplinkFrame{
		PhyPayload: rxpk.Data,
		TxInfo: &gw.UplinkTxInfo{
			Frequency: uint32(rxpk.Freq * 1000000),
		},
		RxInfo: &gw.UplinkRxInfo{
			GatewayId: gatewayID.String(),
			Rssi:      int32(rxpk.RSSI),
			Snr:       float32(rxpk.LSNR),
			RfChain:   uint32(rxpk.RFCh),
			Channel:   uint32(rxpk.Chan),
			Board:     uint32(rxpk.Brd),
			Context:   make([]byte, 4),
			Metadata:  rxpk.Meta,
		},
	}

	// If a Stat is present and it contains a location, immediately set the location for this uplink.
	// This is for example the case of Helium, where the UDP frame contains both a rxpk and stat
	// payload to provide additional gateway context.
	if stat != nil && (stat.Lati != 0 || stat.Long != 0 || stat.Alti != 0) {
		frame.RxInfo.Location = &common.Location{
			Latitude:  stat.Lati,
			Longitude: stat.Long,
			Altitude:  float64(stat.Alti),
			Source:    common.LocationSource_GPS,
		}
	}

	switch rxpk.Stat {
	case 1:
		frame.RxInfo.CrcStatus = gw.CRCStatus_CRC_OK
	case -1:
		frame.RxInfo.CrcStatus = gw.CRCStatus_BAD_CRC
	default:
		frame.RxInfo.CrcStatus = gw.CRCStatus_NO_CRC
	}

	// Context
	binary.BigEndian.PutUint32(frame.RxInfo.Context, rxpk.Tmst)

	// Time.
	if rxpk.Time != nil && !time.Time(*rxpk.Time).IsZero() {
		frame.RxInfo.GwTime = timestamppb.New(time.Time(*rxpk.Time))
	} else if FakeRxInfoTime {
		frame.RxInfo.GwTime = timestamppb.Now()
	}

	// Time since GPS epoch
	if rxpk.Tmms != nil {
		d := time.Duration(*rxpk.Tmms) * time.Millisecond
		frame.RxInfo.TimeSinceGpsEpoch = durationpb.New(d)
	}

	// Plain fine-timestamp (SX1302 / SX1303)
	if rxpk.Tmms != nil && rxpk.FTime != nil {
		d := time.Duration(*rxpk.Tmms) * time.Millisecond

		// take the seconds from the gps time
		d = d - (d % time.Second)
		// add the nanos from the fine-timestamp
		d = d + (time.Duration(*rxpk.FTime) * time.Nanosecond)

		frame.RxInfo.FineTimeSinceGpsEpoch = durationpb.New(d)
	}

	// LoRa data-rate
	if rxpk.DatR.LoRa != "" {
		// parse e.g. SF12BW250 into separate variables
		match := loRaDataRateRegex.FindStringSubmatch(rxpk.DatR.LoRa)
		if len(match) != 3 {
			return &frame, errors.New("backend/semtechudp/packets: could not parse LoRa data-rate")
		}

		// cast variables to ints
		sf, err := strconv.Atoi(match[1])
		if err != nil {
			return &frame, errors.Wrap(err, "backend/semtechudp/packets: could not convert sf to int")
		}

		bw, err := strconv.Atoi(match[2])
		if err != nil {
			return &frame, errors.Wrap(err, "backend/semtechudp/packets: could not parse bandwidth to int")
		}

		cr := gw.CodeRate_CR_UNDEFINED
		switch rxpk.CodR {
		case "4/5":
			cr = gw.CodeRate_CR_4_5
		case "4/6":
			cr = gw.CodeRate_CR_4_6
		case "4/7":
			cr = gw.CodeRate_CR_4_7
		case "4/8":
			cr = gw.CodeRate_CR_4_8
		case "3/8":
			cr = gw.CodeRate_CR_3_8
		case "1/3":
			cr = gw.CodeRate_CR_2_6
		case "2/6":
			cr = gw.CodeRate_CR_2_6
		case "1/4":
			cr = gw.CodeRate_CR_1_4
		case "1/6":
			cr = gw.CodeRate_CR_1_6
		case "5/6":
			cr = gw.CodeRate_CR_5_6
		case "4/5LI":
			cr = gw.CodeRate_CR_LI_4_5
		case "4/6LI":
			cr = gw.CodeRate_CR_LI_4_6
		case "4/8LI":
			cr = gw.CodeRate_CR_LI_4_8
		default:
			return &frame, errors.New(fmt.Sprintf("backend/semtechudp:packets: invalid CodR: %s", rxpk.CodR))
		}

		frame.TxInfo.Modulation = &gw.Modulation{
			Parameters: &gw.Modulation_Lora{
				Lora: &gw.LoraModulationInfo{
					Bandwidth:       uint32(bw) * 1000,
					SpreadingFactor: uint32(sf),
					CodeRate:        cr,
				},
			},
		}
	}

	// LR-FHSS data-rate
	if rxpk.DatR.LRFHSS != "" {
		// parse M0CW137 into CW (OCW) variable
		match := lrFHSSDataRateRegex.FindStringSubmatch(rxpk.DatR.LRFHSS)
		if len(match) != 2 {
			return &frame, errors.New("backend/semtechudp/packets: could not parse LR-FHSS data-rate")
		}

		// cast variable to int
		ocw, err := strconv.Atoi(match[1])
		if err != nil {
			return &frame, errors.Wrap(err, "backend/semtechudp/packets: could not convert cw to int")
		}

		cr := gw.CodeRate_CR_UNDEFINED
		switch rxpk.CodR {
		case "4/5":
			cr = gw.CodeRate_CR_4_5
		case "4/6":
			cr = gw.CodeRate_CR_4_6
		case "4/7":
			cr = gw.CodeRate_CR_4_7
		case "4/8":
			cr = gw.CodeRate_CR_4_8
		case "3/8":
			cr = gw.CodeRate_CR_3_8
		case "1/3":
			cr = gw.CodeRate_CR_2_6
		case "2/6":
			cr = gw.CodeRate_CR_2_6
		case "1/4":
			cr = gw.CodeRate_CR_1_4
		case "1/6":
			cr = gw.CodeRate_CR_1_6
		case "5/6":
			cr = gw.CodeRate_CR_5_6
		case "4/5LI":
			cr = gw.CodeRate_CR_LI_4_5
		case "4/6LI":
			cr = gw.CodeRate_CR_LI_4_6
		case "4/8LI":
			cr = gw.CodeRate_CR_LI_4_8
		default:
			return &frame, errors.New(fmt.Sprintf("backend/semtechudp:packets: invalid CodR: %s", rxpk.CodR))
		}

		frame.TxInfo.Modulation = &gw.Modulation{
			Parameters: &gw.Modulation_LrFhss{
				LrFhss: &gw.LrFhssModulationInfo{
					OperatingChannelWidth: uint32(ocw) * 1000, // kHz -> Hz
					CodeRate:              cr,
					GridSteps:             uint32(rxpk.HPW),
				},
			},
		}
	}

	// FSK data-rate
	if rxpk.DatR.FSK != 0 {
		frame.TxInfo.Modulation = &gw.Modulation{
			Parameters: &gw.Modulation_Fsk{
				Fsk: &gw.FskModulationInfo{
					Datarate: uint32(rxpk.DatR.FSK),
				},
			},
		}
	}

	return &frame, nil
}

// UnmarshalBinary decodes the packet from Semtech UDP binary form.
func (p *PushDataPacket) UnmarshalBinary(data []byte) error {
	if len(data) < 13 {
		return errors.New("backend/semtechudp/packets: at least 13 bytes are expected")
	}
	if data[3] != byte(PushData) {
		return errors.New("backend/semtechudp/packets: identifier mismatch (PUSH_DATA expected)")
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

// PushDataPayload represents the upstream JSON data structure.
type PushDataPayload struct {
	RXPK []RXPK `json:"rxpk,omitempty"`
	Stat *Stat  `json:"stat,omitempty"`
}

// Stat contains the status of the gateway.
type Stat struct {
	Time ExpandedTime      `json:"time"` // UTC 'system' time of the gateway, ISO 8601 'expanded' format (e.g 2014-01-12 08:59:28 GMT)
	Lati float64           `json:"lati"` // GPS latitude of the gateway in degree (float, N is +)
	Long float64           `json:"long"` // GPS latitude of the gateway in degree (float, E is +)
	Alti int32             `json:"alti"` // GPS altitude of the gateway in meter RX (integer)
	RXNb uint32            `json:"rxnb"` // Number of radio packets received (unsigned integer)
	RXOK uint32            `json:"rxok"` // Number of radio packets received with a valid PHY CRC
	RXFW uint32            `json:"rxfw"` // Number of radio packets forwarded (unsigned integer)
	ACKR float64           `json:"ackr"` // Percentage of upstream datagrams that were acknowledged
	DWNb uint32            `json:"dwnb"` // Number of downlink datagrams received (unsigned integer)
	TXNb uint32            `json:"txnb"` // Number of packets emitted (unsigned integer)
	Meta map[string]string `json:"meta"` // Custom meta-data (Optional, not part of PROTOCOL.TXT)
}

// RXPK contain a RF packet and associated metadata.
type RXPK struct {
	Time  *CompactTime      `json:"time"`  // UTC time of pkt RX, us precision, ISO 8601 'compact' format (e.g. 2013-03-31T16:21:17.528002Z)
	Tmms  *int64            `json:"tmms"`  // GPS time of pkt RX, number of milliseconds since 06.Jan.1980
	Tmst  uint32            `json:"tmst"`  // Internal timestamp of "RX finished" event (32b unsigned)
	FTime *uint32           `json:"ftime"` // Fine timestamp, number of nanoseconds since last PPS [0..999999999] (Optional)
	AESK  uint8             `json:"aesk"`  // AES key index used for encrypting fine timestamps
	Chan  uint8             `json:"chan"`  // Concentrator "IF" channel used for RX (unsigned integer)
	RFCh  uint8             `json:"rfch"`  // Concentrator "RF chain" used for RX (unsigned integer)
	Stat  int8              `json:"stat"`  // CRC status: 1 = OK, -1 = fail, 0 = no CRC
	Freq  float64           `json:"freq"`  // RX central frequency in MHz (unsigned float, Hz precision)
	Brd   uint32            `json:"brd"`   // Concentrator board used for RX (unsigned integer)
	RSSI  int16             `json:"rssi"`  // RSSI in dBm (signed integer, 1 dB precision)
	Size  uint16            `json:"size"`  // RF packet payload size in bytes (unsigned integer)
	DatR  DatR              `json:"datr"`  // LoRa datarate identifier (eg. SF12BW500) || FSK datarate (unsigned, in bits per second)
	Modu  string            `json:"modu"`  // Modulation identifier "LORA" or "FSK"
	CodR  string            `json:"codr"`  // LoRa ECC coding rate identifier
	LSNR  float64           `json:"lsnr"`  // Lora SNR ratio in dB (signed float, 0.1 dB precision)
	HPW   uint8             `json:"hpw"`   // LR-FHSS hopping grid number of steps.
	Data  []byte            `json:"data"`  // Base64 encoded RF packet payload, padded
	RSig  []RSig            `json:"rsig"`  // Received signal information, per antenna (Optional)
	Meta  map[string]string `json:"meta"`  // Custom meta-data (Optional, not part of PROTOCOL.TXT)
}

// RSig contains the received signal information per antenna.
type RSig struct {
	Ant   uint8   `json:"ant"`   // Antenna number on which signal has been received
	Chan  uint8   `json:"chan"`  // Concentrator "IF" channel used for RX (unsigned integer)
	RSSIC int16   `json:"rssic"` // RSSI in dBm of the channel (signed integer, 1 dB precision)
	LSNR  float32 `json:"lsnr"`  // Lora SNR ratio in dB (signed float, 0.1 dB precision)
	ETime []byte  `json:"etime"` // Encrypted 'main' fine timestamp, ns precision [0..999999999] (Optional)
}
