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

func TestJoinRequestToProto(t *testing.T) {
	assert := require.New(t)
	b, err := band.GetConfig(band.EU868, false, lorawan.DwellTimeNoLimit)
	assert.NoError(err)

	pTime := timestamppb.New(time.Time(gps.NewTimeFromTimeSinceGPSEpoch(5 * time.Second)))

	jr := JoinRequest{
		RadioMetaData: RadioMetaData{
			DR:        5,
			Frequency: 868100000,
			UpInfo: RadioMetaDataUpInfo{
				RCtx:    1,
				XTime:   2,
				GPSTime: int64(5 * time.Second / time.Microsecond),
				RSSI:    120,
				SNR:     5.5,
			},
		},

		MessageType: JoinRequestMessage,
		MHDR:        0x00,
		JoinEUI:     EUI64{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		DevEUI:      EUI64{0x03, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		DevNonce:    20,
		MIC:         -10,
	}

	pb, err := JoinRequestToProto(b, lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, jr)
	assert.NoError(err)

	assert.Equal(&gw.UplinkFrame{
		PhyPayload: []byte{0x00, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x02, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x03, 0x14, 0x00, 0xf6, 0xff, 0xff, 0xff},
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
			GwTime:            pTime,
			TimeSinceGpsEpoch: durationpb.New(5 * time.Second),
			Rssi:              120,
			Snr:               5.5,
			Context:           []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus:         gw.CRCStatus_CRC_OK,
		},
	}, pb)

}
