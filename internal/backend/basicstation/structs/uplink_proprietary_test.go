package structs

import (
	"testing"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/stretchr/testify/require"
)

func TestUplinkProprietaryFrameToProto(t *testing.T) {
	tests := []struct {
		Name  string
		In    UplinkProprietaryFrame
		Out   gw.UplinkFrame
		Error error
	}{
		{
			Name: "proprietary",
			In: UplinkProprietaryFrame{
				RadioMetaData: RadioMetaData{
					DR:        5,
					Frequency: 868100000,
					UpInfo: RadioMetaDataUpInfo{
						RCtx:  1,
						XTime: 2,
						RSSI:  120,
						SNR:   5.5,
					},
				},
				MessageType: ProprietaryDataFrameMessage,
				FRMPayload:  "01020304",
			},
			Out: gw.UplinkFrame{
				PhyPayload: []byte{0x01, 0x02, 0x03, 0x04},
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
	}

	assert := require.New(t)

	b, err := band.GetConfig(band.EU868, false, lorawan.DwellTimeNoLimit)
	assert.NoError(err)

	for _, tst := range tests {
		assert := require.New(t)

		uf, err := UplinkProprietaryFrameToProto(b, lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, tst.In)
		assert.Equal(tst.Error, err)
		if err != nil {
			return
		}
		assert.Equal(tst.Out, uf)
	}
}
