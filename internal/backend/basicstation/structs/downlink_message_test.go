package structs

import (
	"testing"
	"time"

	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDownlinkFrameFromProto(t *testing.T) {
	delay1 := 1
	dr1 := 1
	dr2 := 2
	dr7 := 7
	freq := uint32(868100000)
	freq2 := uint32(868200000)
	rCtx := uint64(3)
	xTime := uint64(4)
	gpsTime := uint64(time.Second / time.Microsecond)

	tests := []struct {
		Name  string
		In    *gw.DownlinkFrame
		Out   DownlinkFrame
		Error error
	}{
		{
			Name: "Class-A LoRa",
			In: &gw.DownlinkFrame{
				DownlinkId: 1234,
				GatewayId:  "0102030405060708",
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Power:     14,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Lora{
									Lora: &gw.LoraModulationInfo{
										Bandwidth:             125000,
										SpreadingFactor:       10,
										CodeRate:              gw.CodeRate_CR_4_5,
										PolarizationInversion: true,
									},
								},
							},
							Timing: &gw.Timing{
								Parameters: &gw.Timing_Delay{
									Delay: &gw.DelayTimingInfo{
										Delay: durationpb.New(time.Second),
									},
								},
							},
							Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
						},
					},
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868200000,
							Power:     14,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Lora{
									Lora: &gw.LoraModulationInfo{
										Bandwidth:             125000,
										SpreadingFactor:       11,
										CodeRate:              gw.CodeRate_CR_4_5,
										PolarizationInversion: true,
									},
								},
							},
							Timing: &gw.Timing{
								Parameters: &gw.Timing_Delay{
									Delay: &gw.DelayTimingInfo{
										Delay: durationpb.New(time.Second * 2),
									},
								},
							},
							Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
						},
					},
				},
			},
			Out: DownlinkFrame{
				MessageType: DownlinkMessage,
				DevEui:      "01-01-01-01-01-01-01-01",
				DC:          0,
				DIID:        1234,
				Priority:    1,
				PDU:         "01020304",
				RCtx:        &rCtx,
				XTime:       &xTime,
				RxDelay:     &delay1,
				RX1DR:       &dr2,
				RX1Freq:     &freq,
				RX2DR:       &dr1,
				RX2Freq:     &freq2,
			},
		},
		{
			Name: "Class-A FSK",
			In: &gw.DownlinkFrame{
				DownlinkId: 1234,
				GatewayId:  "0102030405060708",
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Power:     14,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Fsk{
									Fsk: &gw.FskModulationInfo{
										Datarate: 50000,
									},
								},
							},
							Timing: &gw.Timing{
								Parameters: &gw.Timing_Delay{
									Delay: &gw.DelayTimingInfo{
										Delay: durationpb.New(time.Second),
									},
								},
							},
							Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
						},
					},
				},
			},
			Out: DownlinkFrame{
				MessageType: DownlinkMessage,
				DevEui:      "01-01-01-01-01-01-01-01",
				DC:          0,
				DIID:        1234,
				Priority:    1,
				PDU:         "01020304",
				RCtx:        &rCtx,
				XTime:       &xTime,
				RxDelay:     &delay1,
				RX1DR:       &dr7,
				RX1Freq:     &freq,
			},
		},
		{
			Name: "Class-B",
			In: &gw.DownlinkFrame{
				DownlinkId: 1234,
				GatewayId:  "0102030405060708",
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Power:     14,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Lora{
									Lora: &gw.LoraModulationInfo{
										Bandwidth:             125000,
										SpreadingFactor:       10,
										CodeRate:              gw.CodeRate_CR_4_5,
										PolarizationInversion: true,
									},
								},
							},
							Timing: &gw.Timing{
								Parameters: &gw.Timing_GpsEpoch{
									GpsEpoch: &gw.GPSEpochTimingInfo{
										TimeSinceGpsEpoch: durationpb.New(time.Second),
									},
								},
							},
							Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
						},
					},
				},
			},
			Out: DownlinkFrame{
				MessageType: DownlinkMessage,
				DevEui:      "01-01-01-01-01-01-01-01",
				DC:          1,
				DIID:        1234,
				Priority:    1,
				PDU:         "01020304",
				RCtx:        &rCtx,
				XTime:       &xTime,
				DR:          &dr2,
				Freq:        &freq,
				GPSTime:     &gpsTime,
			},
		},
		{
			Name: "Class-C",
			In: &gw.DownlinkFrame{
				DownlinkId: 1234,
				GatewayId:  "0102030405060708",
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Power:     14,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Lora{
									Lora: &gw.LoraModulationInfo{
										Bandwidth:             125000,
										SpreadingFactor:       10,
										CodeRate:              gw.CodeRate_CR_4_5,
										PolarizationInversion: true,
									},
								},
							},
							Timing: &gw.Timing{
								Parameters: &gw.Timing_Immediately{
									Immediately: &gw.ImmediatelyTimingInfo{},
								},
							},
							Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
						},
					},
				},
			},
			Out: DownlinkFrame{
				MessageType: DownlinkMessage,
				DevEui:      "01-01-01-01-01-01-01-01",
				DC:          2,
				DIID:        1234,
				Priority:    1,
				PDU:         "01020304",
				RCtx:        &rCtx,
				XTime:       &xTime,
				RX2DR:       &dr2,
				RX2Freq:     &freq,
			},
		},
	}

	assert := require.New(t)
	b, err := band.GetConfig(band.EU868, false, lorawan.DwellTimeNoLimit)
	assert.NoError(err)

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)
			out, err := DownlinkFrameFromProto(b, tst.In)
			assert.Equal(tst.Error, err)
			if err != nil {
				return
			}
			assert.NotNil(out.MuxTime)
			out.MuxTime = nil
			assert.Equal(tst.Out, out)
		})
	}
}
