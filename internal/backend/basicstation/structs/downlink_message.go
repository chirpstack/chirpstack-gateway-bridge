package structs

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-api/go/common"
	"github.com/brocaar/chirpstack-api/go/gw"
	"github.com/brocaar/lorawan/band"
)

// DownlinkMessage implements the downlink message.
type DownlinkFrame struct {
	MessageType MessageType `json:"msgtype"`

	DevEui   string  `json:"DevEui"`
	DC       int     `json:"dC"`
	DIID     uint32  `json:"diid"`
	PDU      string  `json:"pdu"`
	Priority int     `json:"priority"`
	RxDelay  *int    `json:"RxDelay,omitempty"`
	RX1DR    *int    `json:"RX1DR,omitempty"`
	RX1Freq  *uint32 `json:"RX1Freq,omitempty"`
	RX2DR    *int    `json:"RX2DR,omitempty"`
	RX2Freq  *uint32 `json:"RX2Freq,omitempty"`
	DR       *int    `json:"DR,omitempty"`
	Freq     *uint32 `json:"Freq,omitempty"`
	GPSTime  *uint64 `json:"gpstime,omitempty"`
	XTime    *uint64 `json:"xtime,omitempty"`
	RCtx     *uint64 `json:"rctx,omitempty"`
}

// DownlinkFrameFromProto convers the given protobuf message to a DownlinkFrame.
func DownlinkFrameFromProto(loraBand band.Band, pb gw.DownlinkFrame) (DownlinkFrame, error) {

	if pb.TxInfo == nil {
		return DownlinkFrame{}, errors.New("tx_info must not be nil")
	}

	out := DownlinkFrame{
		MessageType: DownlinkMessage,
		Priority:    1,                         // not (yet) available through gw.DownlinkFrame
		DevEui:      "00-00-00-00-00-00-00-00", // set to blank EUI
		DIID:        pb.Token,
		PDU:         hex.EncodeToString(pb.PhyPayload),
	}

	// context
	// depending the scheduling type, there might or might not be a context
	if len(pb.TxInfo.Context) >= 8 {
		var rctx, xtime uint64
		rctx = binary.BigEndian.Uint64(pb.TxInfo.Context[0:8])
		xtime = binary.BigEndian.Uint64(pb.TxInfo.Context[8:16])

		out.RCtx = &rctx
		out.XTime = &xtime
	}

	// get data-rate
	var dr int
	var err error

	switch pb.TxInfo.Modulation {
	case common.Modulation_LORA:
		modInfo := pb.TxInfo.GetLoraModulationInfo()
		if modInfo == nil {
			return out, fmt.Errorf("lora_modulation_info is missing")
		}
		dr, err = loraBand.GetDataRateIndex(false, band.DataRate{
			Modulation:   band.LoRaModulation,
			SpreadFactor: int(modInfo.SpreadingFactor),
			Bandwidth:    int(modInfo.Bandwidth),
		})
		if err != nil {
			return out, errors.Wrap(err, "get data-rate index error")
		}
	case common.Modulation_FSK:
		modInfo := pb.TxInfo.GetFskModulationInfo()
		if modInfo == nil {
			return out, fmt.Errorf("fsk_modulation_info is missing")
		}
		dr, err = loraBand.GetDataRateIndex(false, band.DataRate{
			Modulation: band.FSKModulation,
			Bandwidth:  int(modInfo.Bandwidth),
			BitRate:    int(modInfo.Bitrate),
		})
		if err != nil {
			return out, errors.Wrap(err, "get data-rate index error")
		}
	default:
		return out, fmt.Errorf("unexpected modulation: %s", pb.TxInfo.Modulation)
	}

	switch pb.TxInfo.Timing {
	case gw.DownlinkTiming_IMMEDIATELY:
		out.DC = 2 // Class-C
		out.RX2DR = &dr
		out.RX2Freq = &pb.TxInfo.Frequency
	case gw.DownlinkTiming_DELAY:
		timingInfo := pb.TxInfo.GetDelayTimingInfo()
		if timingInfo == nil {
			return out, errors.New("delay_timing_info must not be nil")
		}
		delayDuration, err := ptypes.Duration(timingInfo.Delay)
		if err != nil {
			return out, errors.Wrap(err, "get delay duration error")
		}
		delay := int(delayDuration / time.Second)

		out.DC = 0 // Class-A
		out.RxDelay = &delay
		out.RX1DR = &dr
		out.RX1Freq = &pb.TxInfo.Frequency
	case gw.DownlinkTiming_GPS_EPOCH:
		timingInfo := pb.TxInfo.GetGpsEpochTimingInfo()
		if timingInfo == nil {
			return out, errors.New("gps_epoch_timing_info must not be nil")
		}
		gpsEpochDuration, err := ptypes.Duration(timingInfo.TimeSinceGpsEpoch)
		if err != nil {
			return out, errors.Wrap(err, "get time since gps epoch error")
		}
		gpsEpoch := uint64(gpsEpochDuration / time.Microsecond)

		out.DC = 1 // Class-B
		out.DR = &dr
		out.Freq = &pb.TxInfo.Frequency
		out.GPSTime = &gpsEpoch

	default:
		return out, fmt.Errorf("unexpected downlink timing: %s", pb.TxInfo.Timing)
	}

	return out, nil
}
