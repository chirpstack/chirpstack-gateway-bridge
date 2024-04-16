package structs

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/brocaar/lorawan/gps"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RadioMetaData contains the radio meta-data.
type RadioMetaData struct {
	DR        int                 `json:"DR"`
	Frequency uint32              `json:"Freq"`
	UpInfo    RadioMetaDataUpInfo `json:"upinfo"`
}

// RadioMetaDataUpInfo contains the radio meta-data uplink info.
type RadioMetaDataUpInfo struct {
	RxTime  float64 `json:"rxtime"`
	RCtx    uint64  `json:"rctx"`
	XTime   uint64  `json:"xtime"`
	GPSTime int64   `json:"gpstime"`
	RSSI    float32 `json:"rssi"`
	SNR     float32 `json:"snr"`
}

// SetRadioMetaDataToProto sets the given parameters to the given protobuf struct.
func SetRadioMetaDataToProto(loraBand band.Band, gatewayID lorawan.EUI64, rmd RadioMetaData, pb *gw.UplinkFrame) error {
	//
	// TxInfo
	//
	dr, err := loraBand.GetDataRate(rmd.DR)
	if err != nil {
		return errors.Wrap(err, "get data-rate error")
	}

	pb.TxInfo = &gw.UplinkTxInfo{
		Frequency: rmd.Frequency,
	}

	switch dr.Modulation {
	case band.LoRaModulation:
		pb.TxInfo.Modulation = &gw.Modulation{
			Parameters: &gw.Modulation_Lora{
				Lora: &gw.LoraModulationInfo{
					Bandwidth:             uint32(dr.Bandwidth) * 1000,
					SpreadingFactor:       uint32(dr.SpreadFactor),
					CodeRate:              gw.CodeRate_CR_4_5,
					PolarizationInversion: false,
				},
			},
		}
	case band.FSKModulation:
		pb.TxInfo.Modulation = &gw.Modulation{
			Parameters: &gw.Modulation_Fsk{
				Fsk: &gw.FskModulationInfo{
					Datarate: uint32(dr.BitRate),
				},
			},
		}
	}

	//
	// RxInfo
	//
	pb.RxInfo = &gw.UplinkRxInfo{
		GatewayId: gatewayID.String(),
		Rssi:      int32(rmd.UpInfo.RSSI),
		Snr:       float32(rmd.UpInfo.SNR),
		CrcStatus: gw.CRCStatus_CRC_OK,
	}

	if rxTime := rmd.UpInfo.RxTime; rxTime != 0 {
		sec, nsec := math.Modf(rmd.UpInfo.RxTime)
		if sec != 0 {
			val := time.Unix(int64(sec), int64(nsec*1e9))
			pb.RxInfo.GwTime = timestamppb.New(val)
		}
	}

	if gpsTime := rmd.UpInfo.GPSTime; gpsTime != 0 {
		gpsTimeDur := time.Duration(gpsTime) * time.Microsecond
		gpsTimeTime := time.Time(gps.NewTimeFromTimeSinceGPSEpoch(gpsTimeDur))

		pb.RxInfo.TimeSinceGpsEpoch = durationpb.New(gpsTimeDur)
		pb.RxInfo.GwTime = timestamppb.New(gpsTimeTime)
	}

	// Context
	pb.RxInfo.Context = make([]byte, 16)
	binary.BigEndian.PutUint64(pb.RxInfo.Context[0:8], uint64(rmd.UpInfo.RCtx))
	binary.BigEndian.PutUint64(pb.RxInfo.Context[8:16], uint64(rmd.UpInfo.XTime))

	return nil
}
