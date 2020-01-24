package structs

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/require"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/brocaar/lorawan/gps"
)

func TestSetRadioMetaDataToProto(t *testing.T) {
	assert := require.New(t)

	timeP, err := ptypes.TimestampProto(time.Time(gps.NewTimeFromTimeSinceGPSEpoch(5 * time.Second)))
	assert.NoError(err)

	tests := []struct {
		Name  string
		In    RadioMetaData
		Out   gw.UplinkFrame
		Error error
	}{
		{
			Name: "LoRa",
			In: RadioMetaData{
				DR:        5,
				Frequency: 868100000,
				UpInfo: RadioMetaDataUpInfo{
					RCtx:  1,
					XTime: 2,
					RSSI:  120,
					SNR:   5.5,
				},
			},
			Out: gw.UplinkFrame{
				TxInfo: &gw.UplinkTXInfo{
					Frequency:  868100000,
					Modulation: common.Modulation_LORA,
					ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
						LoraModulationInfo: &gw.LoRaModulationInfo{
							Bandwidth:       125,
							SpreadingFactor: 7,
							CodeRate:        "4/5",
						},
					},
				},
				RxInfo: &gw.UplinkRXInfo{
					GatewayId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
					Rssi:      120,
					LoraSnr:   5.5,
					Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
					CrcStatus: gw.CRCStatus_CRC_OK,
				},
			},
		},
		{
			Name: "FSK",
			In: RadioMetaData{
				DR:        7,
				Frequency: 868100000,
				UpInfo: RadioMetaDataUpInfo{
					RCtx:  1,
					XTime: 2,
					RSSI:  120,
				},
			},
			Out: gw.UplinkFrame{
				TxInfo: &gw.UplinkTXInfo{
					Frequency:  868100000,
					Modulation: common.Modulation_FSK,
					ModulationInfo: &gw.UplinkTXInfo_FskModulationInfo{
						FskModulationInfo: &gw.FSKModulationInfo{
							Datarate: 50000,
						},
					},
				},
				RxInfo: &gw.UplinkRXInfo{
					GatewayId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
					Rssi:      120,
					Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
					CrcStatus: gw.CRCStatus_CRC_OK,
				},
			},
		},
		{
			Name: "LoRa with GPS time",
			In: RadioMetaData{
				DR:        5,
				Frequency: 868100000,
				UpInfo: RadioMetaDataUpInfo{
					RCtx:    1,
					XTime:   2,
					RSSI:    120,
					SNR:     5.5,
					GPSTime: int64(5 * time.Second / time.Microsecond),
				},
			},
			Out: gw.UplinkFrame{
				TxInfo: &gw.UplinkTXInfo{
					Frequency:  868100000,
					Modulation: common.Modulation_LORA,
					ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
						LoraModulationInfo: &gw.LoRaModulationInfo{
							Bandwidth:       125,
							SpreadingFactor: 7,
							CodeRate:        "4/5",
						},
					},
				},
				RxInfo: &gw.UplinkRXInfo{
					GatewayId:         []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
					Rssi:              120,
					LoraSnr:           5.5,
					Context:           []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
					TimeSinceGpsEpoch: ptypes.DurationProto(5 * time.Second),
					Time:              timeP,
					CrcStatus:         gw.CRCStatus_CRC_OK,
				},
			},
		},
	}

	b, err := band.GetConfig(band.EU868, false, lorawan.DwellTimeNoLimit)
	assert.NoError(err)

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)

			var uf gw.UplinkFrame
			err := SetRadioMetaDataToProto(b, lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, tst.In, &uf)
			assert.Equal(tst.Error, err)
			if err != nil {
				return
			}
			assert.Equal(tst.Out, uf)
		})
	}
}
