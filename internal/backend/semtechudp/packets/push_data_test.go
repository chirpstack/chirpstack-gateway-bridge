package packets

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"

	"github.com/brocaar/loraserver/api/common"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
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
	pbTime, err := ptypes.TimestampProto(now)
	assert.Nil(err)

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
						Long: &long,
						Lati: &lat,
						Alti: &alti,
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
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
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
				GatewayId:           []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Time:                pbTime,
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 2,
				TxPacketsReceived:   5,
				TxPacketsEmitted:    6,
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
	assert := assert.New(t)

	now := time.Now().Truncate(time.Second)
	ctNow := CompactTime(now)
	pbTime, err := ptypes.TimestampProto(now)
	assert.Nil(err)

	tmms := int64(10 * time.Minute / time.Millisecond)

	testTable := []struct {
		PushDataPacket PushDataPacket
		UplinkFrames   []gw.UplinkFrame
	}{
		{
			PushDataPacket: PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				Payload:         PushDataPayload{},
			},
			UplinkFrames: nil,
		},
		{
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
			UplinkFrames: []gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTXInfo{
						Frequency:  868300000,
						Modulation: common.Modulation_LORA,
						ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
							LoraModulationInfo: &gw.LoRaModulationInfo{
								Bandwidth:             500,
								SpreadingFactor:       12,
								CodeRate:              "4/5",
								PolarizationInversion: false,
							},
						},
					},
					RxInfo: &gw.UplinkRXInfo{
						GatewayId:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
						Time:              pbTime,
						TimeSinceGpsEpoch: ptypes.DurationProto(10 * time.Minute),
						Timestamp:         1000000,
						Rssi:              -60,
						LoraSnr:           5.5,
						Channel:           1,
						RfChain:           3,
						Board:             2,
						Antenna:           0,
					},
				},
			},
		},
		{
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
			UplinkFrames: []gw.UplinkFrame{
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTXInfo{
						Frequency:  868300000,
						Modulation: common.Modulation_LORA,
						ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
							LoraModulationInfo: &gw.LoRaModulationInfo{
								Bandwidth:             500,
								SpreadingFactor:       12,
								CodeRate:              "4/5",
								PolarizationInversion: false,
							},
						},
					},
					RxInfo: &gw.UplinkRXInfo{
						GatewayId:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
						Time:              pbTime,
						TimeSinceGpsEpoch: ptypes.DurationProto(10 * time.Minute),
						Timestamp:         1000000,
						Rssi:              -70,
						LoraSnr:           6.6,
						Channel:           9,
						RfChain:           3,
						Board:             2,
						Antenna:           8,
						FineTimestampType: gw.FineTimestampType_ENCRYPTED,
						FineTimestamp: &gw.UplinkRXInfo_EncryptedFineTimestamp{
							EncryptedFineTimestamp: &gw.EncryptedFineTimestamp{
								AesKeyIndex: 7,
								EncryptedNs: []byte{2, 3, 4, 5},
							},
						},
					},
				},
				{
					PhyPayload: []byte{1, 2, 3, 4, 5},
					TxInfo: &gw.UplinkTXInfo{
						Frequency:  868300000,
						Modulation: common.Modulation_LORA,
						ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
							LoraModulationInfo: &gw.LoRaModulationInfo{
								Bandwidth:             500,
								SpreadingFactor:       12,
								CodeRate:              "4/5",
								PolarizationInversion: false,
							},
						},
					},
					RxInfo: &gw.UplinkRXInfo{
						GatewayId:         []byte{1, 2, 3, 4, 5, 6, 7, 8},
						Time:              pbTime,
						TimeSinceGpsEpoch: ptypes.DurationProto(10 * time.Minute),
						Timestamp:         1000000,
						Rssi:              -80,
						LoraSnr:           7.7,
						Channel:           10,
						RfChain:           3,
						Board:             2,
						Antenna:           9,
					},
				},
			},
		},
	}

	for _, test := range testTable {
		f, err := test.PushDataPacket.GetUplinkFrames()
		assert.Nil(err)
		assert.Equal(test.UplinkFrames, f)
	}
}
