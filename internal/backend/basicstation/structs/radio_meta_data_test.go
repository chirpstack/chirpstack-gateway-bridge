package structs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/brocaar/lorawan/gps"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

func TestSetRadioMetaDataToProto(t *testing.T) {
	assert := require.New(t)

	timeP := timestamppb.New(time.Time(gps.NewTimeFromTimeSinceGPSEpoch(5 * time.Second)))

	tests := []struct {
		Name  string
		In    RadioMetaData
		Out   *gw.UplinkFrame
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
			Out: &gw.UplinkFrame{
				TxInfo: &gw.UplinkTxInfo{
					Frequency: 868100000,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:       125000,
								SpreadingFactor: 7,
								CodeRate:        gw.CodeRate_CR_4_5,
							},
						},
					},
				},
				RxInfo: &gw.UplinkRxInfo{
					GatewayId: "0102030405060708",
					Rssi:      120,
					Snr:       5.5,
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
			Out: &gw.UplinkFrame{
				TxInfo: &gw.UplinkTxInfo{
					Frequency: 868100000,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Fsk{
							Fsk: &gw.FskModulationInfo{
								Datarate: 50000,
							},
						},
					},
				},
				RxInfo: &gw.UplinkRxInfo{
					GatewayId: "0102030405060708",
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
			Out: &gw.UplinkFrame{
				TxInfo: &gw.UplinkTxInfo{
					Frequency: 868100000,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:       125000,
								SpreadingFactor: 7,
								CodeRate:        gw.CodeRate_CR_4_5,
							},
						},
					},
				},
				RxInfo: &gw.UplinkRxInfo{
					GatewayId:         "0102030405060708",
					Rssi:              120,
					Snr:               5.5,
					Context:           []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
					TimeSinceGpsEpoch: durationpb.New(5 * time.Second),
					GwTime:            timeP,
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
			assert.Equal(tst.Out, &uf)
		})
	}
}
