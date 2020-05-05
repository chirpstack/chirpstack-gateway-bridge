package packets

import (
	"testing"
	"time"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/require"
)

func TestPullResp(t *testing.T) {
	assert := require.New(t)

	testTable := []struct {
		Bytes          []byte
		PullRespPacket PullRespPacket
	}{
		{
			Bytes: []byte{2, 0, 0, 3, 123, 125},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
			},
		},
		{
			Bytes: []byte{2, 123, 0, 3, 123, 125},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			},
		},
	}

	for _, test := range testTable {
		b, err := test.PullRespPacket.MarshalBinary()
		assert.Nil(err)
		assert.Equal(test.Bytes[0:4], b[0:4])

		var p PullRespPacket
		assert.Nil(p.UnmarshalBinary(test.Bytes))
		assert.Equal(test.PullRespPacket, p)
	}
}

func TestGetPullRespPacket(t *testing.T) {
	timestamp := uint32(2000000)
	timeSinceGPSEpoch := int64(5 * time.Second / time.Millisecond)

	tests := []struct {
		Name           string
		DownlinkFrame  gw.DownlinkFrame
		PullRespPacket PullRespPacket
		Error          error
	}{
		{
			Name: "delay timing - lora",
			DownlinkFrame: gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTXInfo{
							GatewayId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
							Frequency:  868100000,
							Power:      14,
							Modulation: common.Modulation_LORA,
							ModulationInfo: &gw.DownlinkTXInfo_LoraModulationInfo{
								LoraModulationInfo: &gw.LoRaModulationInfo{
									SpreadingFactor:       12,
									Bandwidth:             125,
									PolarizationInversion: true,
									CodeRate:              "4/5",
								},
							},
							Board:   1,
							Antenna: 2,
							Timing:  gw.DownlinkTiming_DELAY,
							TimingInfo: &gw.DownlinkTXInfo_DelayTimingInfo{
								DelayTimingInfo: &gw.DelayTimingInfo{
									Delay: ptypes.DurationProto(time.Second),
								},
							},
							Context: []byte{0x00, 0x0f, 0x42, 0x40},
						},
					},
				},
				Token: 1234,
			},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     1234,
				Payload: PullRespPayload{
					TXPK: TXPK{
						Powe: 14,
						Ant:  2,
						Brd:  1,
						Freq: 868.1,
						Modu: "LORA",
						Tmst: &timestamp,
						DatR: DatR{
							LoRa: "SF12BW125",
						},
						CodR: "4/5",
						IPol: true,
						Size: 4,
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
		},
		{
			Name: "delay timing - fsk",
			DownlinkFrame: gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTXInfo{
							GatewayId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
							Frequency:  868100000,
							Power:      14,
							Modulation: common.Modulation_FSK,
							ModulationInfo: &gw.DownlinkTXInfo_FskModulationInfo{
								FskModulationInfo: &gw.FSKModulationInfo{
									Datarate:           50000,
									FrequencyDeviation: 25000,
								},
							},
							Board:   1,
							Antenna: 2,
							Timing:  gw.DownlinkTiming_DELAY,
							TimingInfo: &gw.DownlinkTXInfo_DelayTimingInfo{
								DelayTimingInfo: &gw.DelayTimingInfo{
									Delay: ptypes.DurationProto(time.Second),
								},
							},
							Context: []byte{0x00, 0x0f, 0x42, 0x40},
						},
					},
				},
				Token: 1234,
			},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     1234,
				Payload: PullRespPayload{
					TXPK: TXPK{
						Powe: 14,
						Ant:  2,
						Brd:  1,
						Freq: 868.1,
						Modu: "FSK",
						Tmst: &timestamp,
						DatR: DatR{
							FSK: 50000,
						},
						FDev: 25000,
						Size: 4,
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
		},
		{
			Name: "immmediately",
			DownlinkFrame: gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTXInfo{
							GatewayId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
							Frequency:  868100000,
							Power:      14,
							Modulation: common.Modulation_LORA,
							ModulationInfo: &gw.DownlinkTXInfo_LoraModulationInfo{
								LoraModulationInfo: &gw.LoRaModulationInfo{
									SpreadingFactor:       12,
									Bandwidth:             125,
									PolarizationInversion: true,
									CodeRate:              "4/5",
								},
							},
							Board:   1,
							Antenna: 2,
							Timing:  gw.DownlinkTiming_IMMEDIATELY,
						},
					},
				},
				Token: 1234,
			},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     1234,
				Payload: PullRespPayload{
					TXPK: TXPK{
						Powe: 14,
						Ant:  2,
						Brd:  1,
						Freq: 868.1,
						Modu: "LORA",
						Imme: true,
						DatR: DatR{
							LoRa: "SF12BW125",
						},
						CodR: "4/5",
						IPol: true,
						Size: 4,
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
		},
		{
			Name: "gps epoch",
			DownlinkFrame: gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTXInfo{
							GatewayId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
							Frequency:  868100000,
							Power:      14,
							Modulation: common.Modulation_LORA,
							ModulationInfo: &gw.DownlinkTXInfo_LoraModulationInfo{
								LoraModulationInfo: &gw.LoRaModulationInfo{
									SpreadingFactor:       12,
									Bandwidth:             125,
									PolarizationInversion: true,
									CodeRate:              "4/5",
								},
							},
							Board:   1,
							Antenna: 2,
							Timing:  gw.DownlinkTiming_GPS_EPOCH,
							TimingInfo: &gw.DownlinkTXInfo_GpsEpochTimingInfo{
								GpsEpochTimingInfo: &gw.GPSEpochTimingInfo{
									TimeSinceGpsEpoch: ptypes.DurationProto(5 * time.Second),
								},
							},
						},
					},
				},
				Token: 1234,
			},
			PullRespPacket: PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     1234,
				Payload: PullRespPayload{
					TXPK: TXPK{
						Powe: 14,
						Ant:  2,
						Brd:  1,
						Freq: 868.1,
						Tmms: &timeSinceGPSEpoch,
						Modu: "LORA",
						DatR: DatR{
							LoRa: "SF12BW125",
						},
						CodR: "4/5",
						IPol: true,
						Size: 4,
						Data: []byte{0x01, 0x02, 0x03, 0x04},
					},
				},
			},
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)

			resp, err := GetPullRespPacket(ProtocolVersion2, 1234, tst.DownlinkFrame, 0)
			assert.Equal(tst.Error, err)
			if err != nil {
				return
			}

			assert.Equal(tst.PullRespPacket, resp)
		})
	}
}
