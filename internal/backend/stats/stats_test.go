package stats

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

func TestStats(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		assert := require.New(t)

		c := NewCollector()
		stats := c.ExportStats()
		assert.True(proto.Equal(stats, &gw.GatewayStats{}))
	})

	t.Run("Uplink", func(t *testing.T) {
		t.Run("LoRa", func(t *testing.T) {
			assert := require.New(t)

			uf := gw.UplinkFrame{
				TxInfo: &gw.UplinkTxInfo{
					Frequency: 868100000,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:             125000,
								SpreadingFactor:       7,
								CodeRate:              gw.CodeRate_CR_4_5,
								PolarizationInversion: false,
							},
						},
					},
				},
			}

			c := NewCollector()
			c.CountUplink(&uf)
			stats := c.ExportStats()

			assert.True(proto.Equal(&gw.GatewayStats{
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 1,
				RxPacketsPerFrequency: map[uint32]uint32{
					868100000: 1,
				},
				RxPacketsPerModulation: []*gw.PerModulationCount{
					{
						Count: 1,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             125000,
									SpreadingFactor:       7,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
				},
			}, stats))
		})

		t.Run("FSK", func(t *testing.T) {
			assert := require.New(t)

			uf := gw.UplinkFrame{
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
			}

			c := NewCollector()
			c.CountUplink(&uf)
			stats := c.ExportStats()

			assert.True(proto.Equal(&gw.GatewayStats{
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 1,
				RxPacketsPerFrequency: map[uint32]uint32{
					868100000: 1,
				},
				RxPacketsPerModulation: []*gw.PerModulationCount{
					{
						Count: 1,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Fsk{
								Fsk: &gw.FskModulationInfo{
									Datarate: 50000,
								},
							},
						},
					},
				},
			}, stats))
		})

		t.Run("LR-FHSS", func(t *testing.T) {
			assert := require.New(t)

			uf := gw.UplinkFrame{
				TxInfo: &gw.UplinkTxInfo{
					Frequency: 868100000,
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
			}

			c := NewCollector()
			c.CountUplink(&uf)
			stats := c.ExportStats()

			assert.True(proto.Equal(&gw.GatewayStats{
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 1,
				RxPacketsPerFrequency: map[uint32]uint32{
					868100000: 1,
				},
				RxPacketsPerModulation: []*gw.PerModulationCount{
					{
						Count: 1,
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
				},
			}, stats))
		})
	})

	t.Run("Downlink", func(t *testing.T) {
		ack := gw.DownlinkTxAck{
			Items: []*gw.DownlinkTxAckItem{
				{
					Status: gw.TxAckStatus_COLLISION_BEACON,
				},
				{
					Status: gw.TxAckStatus_OK,
				},
			},
		}

		t.Run("LoRa", func(t *testing.T) {
			assert := require.New(t)

			df := gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868200000,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Lora{
									Lora: &gw.LoraModulationInfo{
										Bandwidth:             125000,
										SpreadingFactor:       7,
										CodeRate:              gw.CodeRate_CR_4_5,
										PolarizationInversion: false,
									},
								},
							},
						},
					},
					{
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Lora{
									Lora: &gw.LoraModulationInfo{
										Bandwidth:             125000,
										SpreadingFactor:       7,
										CodeRate:              gw.CodeRate_CR_4_5,
										PolarizationInversion: false,
									},
								},
							},
						},
					},
				},
			}

			c := NewCollector()
			c.CountDownlink(&df, &ack)
			stats := c.ExportStats()

			assert.True(proto.Equal(&gw.GatewayStats{
				TxPacketsReceived: 1,
				TxPacketsEmitted:  1,
				TxPacketsPerFrequency: map[uint32]uint32{
					868100000: 1,
				},
				TxPacketsPerModulation: []*gw.PerModulationCount{
					{
						Count: 1,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Lora{
								Lora: &gw.LoraModulationInfo{
									Bandwidth:             125000,
									SpreadingFactor:       7,
									CodeRate:              gw.CodeRate_CR_4_5,
									PolarizationInversion: false,
								},
							},
						},
					},
				},
				TxPacketsPerStatus: map[string]uint32{
					"OK":               1,
					"COLLISION_BEACON": 1,
				},
			}, stats))
		})

		t.Run("FSK", func(t *testing.T) {
			assert := require.New(t)

			df := gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868200000,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Fsk{
									Fsk: &gw.FskModulationInfo{
										Datarate: 50000,
									},
								},
							},
						},
					},
					{
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Fsk{
									Fsk: &gw.FskModulationInfo{
										Datarate: 50000,
									},
								},
							},
						},
					},
				},
			}

			c := NewCollector()
			c.CountDownlink(&df, &ack)
			stats := c.ExportStats()

			assert.True(proto.Equal(&gw.GatewayStats{
				TxPacketsReceived: 1,
				TxPacketsEmitted:  1,
				TxPacketsPerFrequency: map[uint32]uint32{
					868100000: 1,
				},
				TxPacketsPerModulation: []*gw.PerModulationCount{
					{
						Count: 1,
						Modulation: &gw.Modulation{
							Parameters: &gw.Modulation_Fsk{
								Fsk: &gw.FskModulationInfo{
									Datarate: 50000,
								},
							},
						},
					},
				},
				TxPacketsPerStatus: map[string]uint32{
					"OK":               1,
					"COLLISION_BEACON": 1,
				},
			}, stats))
		})
	})
}
