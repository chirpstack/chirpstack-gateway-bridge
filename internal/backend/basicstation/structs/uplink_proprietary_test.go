package structs

import (
	"testing"

	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
	"github.com/stretchr/testify/require"
)

func TestUplinkProprietaryFrameToProto(t *testing.T) {
	tests := []struct {
		Name  string
		In    UplinkProprietaryFrame
		Out   *gw.UplinkFrame
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
			Out: &gw.UplinkFrame{
				PhyPayload: []byte{0x01, 0x02, 0x03, 0x04},
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
