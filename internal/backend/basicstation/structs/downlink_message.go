package structs

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/pkg/errors"

	"github.com/brocaar/lorawan/band"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

// DownlinkFrame implements the downlink message.
type DownlinkFrame struct {
	MessageType MessageType `json:"msgtype"`

	DevEui   string   `json:"DevEui"`
	DC       int      `json:"dC"`
	DIID     uint32   `json:"diid"`
	PDU      string   `json:"pdu"`
	Priority int      `json:"priority"`
	RxDelay  *int     `json:"RxDelay,omitempty"`
	RX1DR    *int     `json:"RX1DR,omitempty"`
	RX1Freq  *uint32  `json:"RX1Freq,omitempty"`
	RX2DR    *int     `json:"RX2DR,omitempty"`
	RX2Freq  *uint32  `json:"RX2Freq,omitempty"`
	DR       *int     `json:"DR,omitempty"`
	Freq     *uint32  `json:"Freq,omitempty"`
	GPSTime  *uint64  `json:"gpstime,omitempty"`
	XTime    *uint64  `json:"xtime,omitempty"`
	RCtx     *uint64  `json:"rctx,omitempty"`
	MuxTime  *float64 `json:"MuxTime,omitempty"`
}

// DownlinkFrameFromProto convers the given protobuf message to a DownlinkFrame.
func DownlinkFrameFromProto(loraBand band.Band, pb *gw.DownlinkFrame) (DownlinkFrame, error) {
	if len(pb.Items) == 0 {
		return DownlinkFrame{}, errors.New("items must contain at least one item")
	}

	// MuxTime
	muxTime := float64(time.Now().UnixMicro()) / 1000000

	// We assume this is for RX1
	item := pb.Items[0]

	out := DownlinkFrame{
		MessageType: DownlinkMessage,
		Priority:    1,                         // not (yet) available through gw.DownlinkFrame
		DevEui:      "01-01-01-01-01-01-01-01", // set to fake DevEUI (setting it to 0 causes the BasicStation to not send acks, see https://github.com/lorabasics/basicstation/issues/71).
		DIID:        pb.DownlinkId,
		PDU:         hex.EncodeToString(item.PhyPayload),
		MuxTime:     &muxTime,
	}

	// context
	// depending the scheduling type, there might or might not be a context
	if len(item.GetTxInfo().Context) >= 8 {
		var rctx, xtime uint64
		rctx = binary.BigEndian.Uint64(item.GetTxInfo().Context[0:8])
		xtime = binary.BigEndian.Uint64(item.GetTxInfo().Context[8:16])

		out.RCtx = &rctx
		out.XTime = &xtime
	}

	// get data-rate
	var dr int
	var err error

	modulation := item.GetTxInfo().GetModulation()
	if lora := modulation.GetLora(); lora != nil {
		dr, err = loraBand.GetDataRateIndex(false, band.DataRate{
			Modulation:   band.LoRaModulation,
			SpreadFactor: int(lora.SpreadingFactor),
			Bandwidth:    int(lora.Bandwidth / 1000),
		})
		if err != nil {
			return out, errors.Wrap(err, "get data-rate index error")
		}
	}

	if fsk := modulation.GetFsk(); fsk != nil {
		dr, err = loraBand.GetDataRateIndex(false, band.DataRate{
			Modulation: band.FSKModulation,
			BitRate:    int(fsk.Datarate),
		})
		if err != nil {
			return out, errors.Wrap(err, "get data-rate index error")
		}
	}

	timing := item.GetTxInfo().GetTiming()
	if immediately := timing.GetImmediately(); immediately != nil {
		out.DC = 2 // Class-C
		out.RX2DR = &dr
		out.RX2Freq = &item.GetTxInfo().Frequency
	}

	if delay := timing.GetDelay(); delay != nil {
		delay := int(delay.Delay.AsDuration() / time.Second)

		out.DC = 0 // Class-A
		out.RxDelay = &delay
		out.RX1DR = &dr
		out.RX1Freq = &item.GetTxInfo().Frequency
	}

	if gpsEpoch := timing.GetGpsEpoch(); gpsEpoch != nil {
		gpsEpoch := uint64(gpsEpoch.TimeSinceGpsEpoch.AsDuration() / time.Microsecond)

		out.DC = 1 // Class-B
		out.DR = &dr
		out.Freq = &item.GetTxInfo().Frequency
		out.GPSTime = &gpsEpoch
	}

	// We assume this is the RX2.
	if len(pb.Items) == 2 {
		item := pb.Items[1]

		if delay := item.GetTxInfo().GetTiming().GetDelay(); delay != nil {
			modulation := item.GetTxInfo().GetModulation()

			if lora := modulation.GetLora(); lora != nil {
				dr, err := loraBand.GetDataRateIndex(false, band.DataRate{
					Modulation:   band.LoRaModulation,
					SpreadFactor: int(lora.SpreadingFactor),
					Bandwidth:    int(lora.Bandwidth / 1000),
				})
				if err != nil {
					return out, errors.Wrap(err, "get data-rate index error")
				}

				out.RX2Freq = &item.GetTxInfo().Frequency
				out.RX2DR = &dr
			}

			if fsk := modulation.GetFsk(); fsk != nil {
				dr, err := loraBand.GetDataRateIndex(false, band.DataRate{
					Modulation: band.FSKModulation,
					BitRate:    int(fsk.Datarate),
				})
				if err != nil {
					return out, errors.Wrap(err, "get data-rate index error")
				}

				out.RX2Freq = &item.GetTxInfo().Frequency
				out.RX2DR = &dr
			}
		}
	}

	return out, nil
}
