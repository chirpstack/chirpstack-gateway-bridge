package structs

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/brocaar/lorawan/gps"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
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

	pb.TxInfo = &gw.UplinkTXInfo{
		Frequency: rmd.Frequency,
	}

	switch dr.Modulation {
	case band.LoRaModulation:
		pb.TxInfo.Modulation = common.Modulation_LORA
		pb.TxInfo.ModulationInfo = &gw.UplinkTXInfo_LoraModulationInfo{
			LoraModulationInfo: &gw.LoRaModulationInfo{
				Bandwidth:             uint32(dr.Bandwidth),
				SpreadingFactor:       uint32(dr.SpreadFactor),
				CodeRate:              "4/5",
				PolarizationInversion: false,
			},
		}
	case band.FSKModulation:
		pb.TxInfo.Modulation = common.Modulation_FSK
		pb.TxInfo.ModulationInfo = &gw.UplinkTXInfo_FskModulationInfo{
			FskModulationInfo: &gw.FSKModulationInfo{
				Datarate: uint32(dr.BitRate),
			},
		}
	}

	//
	// RxInfo
	//
	pb.RxInfo = &gw.UplinkRXInfo{
		GatewayId: gatewayID[:],
		Rssi:      int32(rmd.UpInfo.RSSI),
		LoraSnr:   float64(rmd.UpInfo.SNR),
		CrcStatus: gw.CRCStatus_CRC_OK,
	}

	if rxTime := rmd.UpInfo.RxTime; rxTime != 0 {
		sec, nsec := math.Modf(rmd.UpInfo.RxTime)
		if sec != 0 {
			val := time.Unix(int64(sec), int64(nsec))
			pb.RxInfo.Time, err = ptypes.TimestampProto(val)
			if err != nil {
				return errors.Wrap(err, "rxtime/timestamp proto error")
			}
		}
	}

	if gpsTime := rmd.UpInfo.GPSTime; gpsTime != 0 {
		gpsTimeDur := time.Duration(gpsTime) * time.Microsecond
		gpsTimeTime := time.Time(gps.NewTimeFromTimeSinceGPSEpoch(gpsTimeDur))

		pb.RxInfo.TimeSinceGpsEpoch = ptypes.DurationProto(gpsTimeDur)
		pb.RxInfo.Time, err = ptypes.TimestampProto(gpsTimeTime)
		if err != nil {
			return errors.Wrap(err, "GPSTime/timestamp proto error")
		}

	}

	// Context
	pb.RxInfo.Context = make([]byte, 16)
	binary.BigEndian.PutUint64(pb.RxInfo.Context[0:8], uint64(rmd.UpInfo.RCtx))
	binary.BigEndian.PutUint64(pb.RxInfo.Context[8:16], uint64(rmd.UpInfo.XTime))

	return nil
}
