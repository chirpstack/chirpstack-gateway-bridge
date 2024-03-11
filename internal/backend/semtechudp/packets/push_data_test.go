package packets

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/brocaar/lorawan"
	"github.com/chirpstack/chirpstack/api/go/v4/common"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

func TestPushDataTest(t *testing.T) {
	assert := assert.New(t)

	testTable := []struct {
		Bytes          []byte
		PushDataPacket PushDataPacket
	}{
		{
			PushDataPacket: PushDataPacket{ProtocolVersion: ProtocolVersion2},
			Bytes:          []byte{2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 123, 125},
		},
		{
			Bytes: []byte{2, 123, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 123, 125},
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			},
		},
	}

	for _, test := range testTable {
		b, err := test.PushDataPacket.MarshalBinary()
		assert.Nil(err)
		assert.Equal(test.Bytes, b)

		var pdp PushDataPacket
		assert.Nil(pdp.UnmarshalBinary(test.Bytes))
		assert.Equal(test.PushDataPacket, pdp)
	}
}

func TestGetGatewayStats(t *testing.T) {
	assert := assert.New(t)

	lat := float64(1.123)
	long := float64(2.123)
	alti := int32(33)

	now := time.Now().Truncate(time.Second)
	ecNow := ExpandedTime(now)
	pbTime := timestamppb.New(now)

	testTable := []struct {
		PushDataPacket PushDataPacket
		GatewayStats   *gw.GatewayStats
	}{
		{
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				Payload:         PushDataPayload{},
			},
			GatewayStats: nil,
		},
		{
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: PushDataPayload{
					Stat: &Stat{
						Time: ecNow,
						Long: long,
						Lati: lat,
						Alti: alti,
						RXNb: 1,
						RXOK: 2,
						RXFW: 3,
						ACKR: 4,
						DWNb: 5,
						TXNb: 6,
					},
				},
			},
			GatewayStats: &gw.GatewayStats{
				GatewayId: "0102030405060708",
				Time:      pbTime,
				Location: &common.Location{
					Latitude:  1.123,
					Longitude: 2.123,
					Altitude:  33,
					Source:    common.LocationSource_GPS,
				},
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 2,
				TxPacketsReceived:   5,
				TxPacketsEmitted:    6,
			},
		},
		{
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: PushDataPayload{
					Stat: &Stat{
						Time: ecNow,
						RXNb: 1,
						RXOK: 2,
						RXFW: 3,
						ACKR: 4,
						DWNb: 5,
						TXNb: 6,
					},
				},
			},
			GatewayStats: &gw.GatewayStats{
				GatewayId:           "0102030405060708",
				Time:                pbTime,
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 2,
				TxPacketsReceived:   5,
				TxPacketsEmitted:    6,
			},
		},
		{
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: PushDataPayload{
					Stat: &Stat{
						Time: ecNow,
						RXNb: 1,
						RXOK: 2,
						RXFW: 3,
						ACKR: 4,
						DWNb: 5,
						TXNb: 6,
						Meta: map[string]string{
							"gateway_name": "test-gateway",
						},
					},
				},
			},
			GatewayStats: &gw.GatewayStats{
				GatewayId:           "0102030405060708",
				Time:                pbTime,
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 2,
				TxPacketsReceived:   5,
				TxPacketsEmitted:    6,
				Metadata: map[string]string{
					"gateway_name": "test-gateway",
				},
			},
		},
	}

	for _, test := range testTable {
		s, err := test.PushDataPacket.GetGatewayStats()
		assert.Nil(err)
		assert.Equal(test.GatewayStats, s)
	}
}

func TestGetUplinkFrame(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ctNow := CompactTime(now)
	pbTime := timestamppb.New(now)

	tmms := int64(10 * time.Minute / time.Millisecond)
	ftime := uint32(999999999)

	ft := durationpb.New((time.Duration(tmms) * time.Millisecond) + (time.Duration(ftime) * time.Nanosecond))

	testTable := []struct {
		Name           string
		PushDataPacket PushDataPacket
		UplinkFrames   []*gw.UplinkFrame
	}{
		{
			Name: "no payload",
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				Payload:         PushDataPayload{},
			},
			UplinkFrames: nil,
		},
		{
			Name: "uplink with invalid crc",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Time: &ctNow,
							Tmst: 1000000,
							Freq: 868.3,
							Brd:  2,
							Chan: 1,
							RFCh: 3,
							Stat: -1,
							Modu: "LORA",
							DatR: DatR{LoRa: "SF12BW500"},
							CodR: "4/5",
							RSSI: -60,
							LSNR: 5.5,
							Size: 5,
							Data: []byte{1, 2, 3, 4, 5},
						},
					},
				},
			},
		},
		{
			Name: "uplink with gps time",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Time: &ctNow,
							Tmms: &tmms,
							Tmst: 1000000,
							Freq: 868.3,
							Brd:  2,
							Chan: 1,
							RFCh: 3,
							Stat: 1,
							Modu: "LORA",
							DatR: DatR{LoRa: "SF12BW500"},
							CodR: "4/5",
							RSSI: -60,
							LSNR: 5.5,
							Size: 5,
							Data: []byte{1, 2, 3, 4, 5},
						},
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             500000,
									SpreadingFactor:       12,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId:         "0102030405060708",
						GwTime:            pbTime,
						TimeSinceGpsEpoch: durationpb.New(10 * time.Minute),
						Rssi:              -60,
						Snr:               5.5,
						Channel:           1,
						RfChain:           3,
						Board:             2,
						Antenna:           0,
						Context:           []byte{0x00, 0x0f, 0x42, 0x40},
						CrcStatus:         gw.CRCStatus_CRC_OK,
					},
				},
			},
		},
		{
			Name: "uplink with multiple antennas",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Time: &ctNow,
							Tmms: &tmms,
							Tmst: 1000000,
							Freq: 868.3,
							AESK: 7,
							Brd:  2,
							Chan: 1,
							RFCh: 3,
							Stat: 1,
							Modu: "LORA",
							DatR: DatR{LoRa: "SF12BW500"},
							CodR: "4/5",
							RSSI: -60,
							LSNR: 5.5,
							Size: 5,
							Data: []byte{1, 2, 3, 4, 5},
							RSig: []RSig{
								{
									Ant:   8,
									Chan:  9,
									LSNR:  6.6,
									ETime: []byte{2, 3, 4, 5},
									RSSIC: -70,
								},
								{
									Ant:   9,
									Chan:  10,
									LSNR:  7.7,
									RSSIC: -80,
								},
							},
						},
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             500000,
									SpreadingFactor:       12,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId:         "0102030405060708",
						GwTime:            pbTime,
						TimeSinceGpsEpoch: durationpb.New(10 * time.Minute),
						Rssi:              -70,
						Snr:               6.6,
						Channel:           9,
						RfChain:           3,
						Board:             2,
						Antenna:           8,
						Context:           []byte{0x00, 0x0f, 0x42, 0x40},
						CrcStatus:         gw.CRCStatus_CRC_OK,
					},
				},
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             500000,
									SpreadingFactor:       12,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId:         "0102030405060708",
						GwTime:            pbTime,
						TimeSinceGpsEpoch: durationpb.New(10 * time.Minute),
						Rssi:              -80,
						Snr:               7.7,
						Channel:           10,
						RfChain:           3,
						Board:             2,
						Antenna:           9,
						Context:           []byte{0x00, 0x0f, 0x42, 0x40},
						CrcStatus:         gw.CRCStatus_CRC_OK,
					},
				},
			},
		},
		{
			Name: "LR-FHSS modulation",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Tmst: 1000000,
							Freq: 868.3,
							Stat: 1,
							Modu: "LR-FHSS",
							DatR: DatR{LRFHSS: "M0CW137"},
							CodR: "4/6",
							HPW:  8,
							Size: 5,
							Data: []byte{1, 2, 3, 4, 5},
							RSig: []RSig{
								{
									RSSIC: -74,
								},
							},
						},
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_LrFhss{
								LrFhss: &gw.LrFhssModulationInfo{
									OperatingChannelWidth: 137000,
									CodeRate:              gw.CodeRate_CR_4_6,
									GridSteps:             8,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId: "0102030405060708",
						Rssi:      -74,
						Context:   []byte{0x00, 0x0f, 0x42, 0x40},
						CrcStatus: gw.CRCStatus_CRC_OK,
					},
				},
			},
		},
		{
			Name: "With plain fine-timestamp (ftime)",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Time:  &ctNow,
							Tmms:  &tmms,
							FTime: &ftime,
							Tmst:  1000000,
							Freq:  868.3,
							Brd:   2,
							Chan:  1,
							RFCh:  3,
							Stat:  1,
							Modu:  "LORA",
							DatR:  DatR{LoRa: "SF12BW500"},
							CodR:  "4/5",
							RSSI:  -60,
							LSNR:  5.5,
							Size:  5,
							Data:  []byte{1, 2, 3, 4, 5},
						},
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             500000,
									SpreadingFactor:       12,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId:             "0102030405060708",
						GwTime:                pbTime,
						TimeSinceGpsEpoch:     durationpb.New(10 * time.Minute),
						FineTimeSinceGpsEpoch: ft,
						Rssi:                  -60,
						Snr:                   5.5,
						Channel:               1,
						RfChain:               3,
						Board:                 2,
						Antenna:               0,
						Context:               []byte{0x00, 0x0f, 0x42, 0x40},
						CrcStatus:             gw.CRCStatus_CRC_OK,
					},
				},
			},
		},
		{
			Name: "uplink with meta",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Time: &ctNow,
							Tmst: 1000000,
							Freq: 868.3,
							Brd:  2,
							Chan: 1,
							RFCh: 3,
							Stat: 1,
							Modu: "LORA",
							DatR: DatR{LoRa: "SF12BW500"},
							CodR: "4/5",
							RSSI: -60,
							LSNR: 5.5,
							Size: 5,
							Data: []byte{1, 2, 3, 4, 5},
							Meta: map[string]string{
								"gateway_name": "test-gateway",
							},
						},
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             500000,
									SpreadingFactor:       12,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId: "0102030405060708",
						GwTime:    pbTime,
						Rssi:      -60,
						Snr:       5.5,
						Channel:   1,
						RfChain:   3,
						Board:     2,
						Antenna:   0,
						Context:   []byte{0x00, 0x0f, 0x42, 0x40},
						Metadata: map[string]string{
							"gateway_name": "test-gateway",
						},
						CrcStatus: gw.CRCStatus_CRC_OK,
					},
				},
			},
		},
		{
			Name: "uplink with stat (with location)",
			PushDataPacket: PushDataPacket{
				GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				ProtocolVersion: ProtocolVersion2,
				Payload: PushDataPayload{
					RXPK: []RXPK{
						{
							Time: &ctNow,
							Tmst: 1000000,
							Freq: 868.3,
							Brd:  2,
							Chan: 1,
							RFCh: 3,
							Stat: 1,
							Modu: "LORA",
							DatR: DatR{LoRa: "SF12BW500"},
							CodR: "4/5",
							RSSI: -60,
							LSNR: 5.5,
							Size: 5,
							Data: []byte{1, 2, 3, 4, 5},
						},
					},
					Stat: &Stat{
						Lati: 1.1,
						Long: 1.2,
						Alti: 10,
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868300000,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             500000,
									SpreadingFactor:       12,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
					RxInfo: &gw.UplinkRxInfo{
						GatewayId: "0102030405060708",
						GwTime:    pbTime,
						Rssi:      -60,
						Snr:       5.5,
						Channel:   1,
						RfChain:   3,
						Board:     2,
						Antenna:   0,
						Context:   []byte{0x00, 0x0f, 0x42, 0x40},
						CrcStatus: gw.CRCStatus_CRC_OK,
						Location: &common.Location{
							Latitude:  1.1,
							Longitude: 1.2,
							Altitude:  10,
							Source:    common.LocationSource_GPS,
						},
					},
				},
			},
		},
	}

	for _, test := range testTable {
		t.Run(test.Name, func(t *testing.T) {
			assert := require.New(t)
			f, err := test.PushDataPacket.GetUplinkFrames(false, false)
			assert.Nil(err)

			for _, ff := range f {
				assert.NotEqual(ff.RxInfo.UplinkId, 0)
				ff.RxInfo.UplinkId = 0
			}

			assert.Equal(test.UplinkFrames, f)
		})
	}
}
