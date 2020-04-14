package structs

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
)

func TestRouterConfig(t *testing.T) {
	tests := []struct {
		Name         string
		Region       band.Name
		NetIDs       []lorawan.NetID
		JoinEUIs     [][2]lorawan.EUI64
		FrequencyMin uint32
		FrequencyMax uint32

		Concentrators []config.BasicStationConcentrator

		ExpectedRouterConfig RouterConfig
		ExpectedError        error
	}{
		{
			Name:         "EU868 3 channels",
			Region:       "EU868",
			NetIDs:       []lorawan.NetID{{0x01, 0x02, 0x03}},
			JoinEUIs:     [][2]lorawan.EUI64{{{}, {0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}},
			FrequencyMin: 863000000,
			FrequencyMax: 870000000,
			Concentrators: []config.BasicStationConcentrator{
				{
					MultiSF: config.BasicStationConcentratorMultiSF{
						Frequencies: []uint32{868100000, 868300000, 868500000},
					},
				},
			},

			ExpectedRouterConfig: RouterConfig{
				MessageType: RouterConfigMessage,
				NetID:       []uint32{66051},
				JoinEui:     [][]uint64{{0, 72623859790382856}},
				Region:      "EU863",
				HWSpec:      "sx1301/1",
				FreqRange:   []uint32{863000000, 870000000},
				DRs: [][]int{
					{12, 125, 0},
					{11, 125, 0},
					{10, 125, 0},
					{9, 125, 0},
					{8, 125, 0},
					{7, 125, 0},
					{7, 250, 0},
					{0, 0, 0}, // FSK
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
				},
				SX1301Conf: []SX1301Conf{
					{
						Radio0: SX1301ConfRadio{
							Enable: true,
							Freq:   868500000,
						},
						ChanMultiSF0: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     -400000,
						},
						ChanMultiSF1: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     -200000,
						},
						ChanMultiSF2: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     0,
						},
					},
				},
			},
		},
		{
			Name:         "EU868 8 + 2 channels",
			Region:       "EU868",
			NetIDs:       []lorawan.NetID{{0x01, 0x02, 0x03}},
			JoinEUIs:     [][2]lorawan.EUI64{{{}, {0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}},
			FrequencyMin: 863000000,
			FrequencyMax: 870000000,
			Concentrators: []config.BasicStationConcentrator{
				{
					MultiSF: config.BasicStationConcentratorMultiSF{
						Frequencies: []uint32{868100000, 868300000, 868500000, 867100000, 867300000, 867500000, 867700000, 867900000},
					},
					LoRaSTD: config.BasicStationConcentratorLoRaSTD{
						Frequency:       868300000,
						Bandwidth:       250000,
						SpreadingFactor: 7,
					},
					FSK: config.BasicStationConcentratorFSK{
						Frequency: 868800000,
					},
				},
			},

			ExpectedRouterConfig: RouterConfig{
				MessageType: RouterConfigMessage,
				NetID:       []uint32{66051},
				JoinEui:     [][]uint64{{0, 72623859790382856}},
				Region:      "EU863",
				HWSpec:      "sx1301/1",
				FreqRange:   []uint32{863000000, 870000000},
				DRs: [][]int{
					{12, 125, 0},
					{11, 125, 0},
					{10, 125, 0},
					{9, 125, 0},
					{8, 125, 0},
					{7, 125, 0},
					{7, 250, 0},
					{0, 0, 0}, // FSK
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
				},
				SX1301Conf: []SX1301Conf{
					{
						Radio0: SX1301ConfRadio{
							Enable: true,
							Freq:   867500000,
						},
						Radio1: SX1301ConfRadio{
							Enable: true,
							Freq:   868500000,
						},
						ChanFSK: SX1301ConfChanFSK{
							Enable: true,
						},
						ChanLoRaStd: SX1301ConfChanLoRaStd{
							Enable:          true,
							Radio:           1,
							IF:              -200000,
							Bandwidth:       250000,
							SpreadingFactor: 7,
						},
						ChanMultiSF0: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     -400000,
						},
						ChanMultiSF1: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     -200000,
						},
						ChanMultiSF2: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     0,
						},
						ChanMultiSF3: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     200000,
						},
						ChanMultiSF4: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     400000,
						},
						ChanMultiSF5: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  1,
							IF:     -400000,
						},
						ChanMultiSF6: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  1,
							IF:     -200000,
						},
						ChanMultiSF7: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  1,
							IF:     0,
						},
					},
				},
			},
		},
		{
			Name:         "US915 0 - 7 + 64",
			Region:       "US915",
			NetIDs:       []lorawan.NetID{{0x01, 0x02, 0x03}},
			JoinEUIs:     [][2]lorawan.EUI64{{{}, {0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}},
			FrequencyMin: 902000000,
			FrequencyMax: 928000000,
			Concentrators: []config.BasicStationConcentrator{
				{
					MultiSF: config.BasicStationConcentratorMultiSF{
						Frequencies: []uint32{902300000, 902500000, 902700000, 902900000, 903100000, 903300000, 903500000, 903700000},
					},
					LoRaSTD: config.BasicStationConcentratorLoRaSTD{
						Frequency:       903000000,
						Bandwidth:       500000,
						SpreadingFactor: 8,
					},
				},
			},

			ExpectedRouterConfig: RouterConfig{
				MessageType: RouterConfigMessage,
				NetID:       []uint32{66051},
				JoinEui:     [][]uint64{{0, 72623859790382856}},
				Region:      "US902",
				HWSpec:      "sx1301/1",
				FreqRange:   []uint32{902000000, 928000000},
				DRs: [][]int{
					{10, 125, 0},
					{9, 125, 0},
					{8, 125, 0},
					{7, 125, 0},
					{8, 500, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{-1, 0, 0},
					{12, 500, 1},
					{11, 500, 1},
					{10, 500, 1},
					{9, 500, 1},
					{8, 500, 0}, // known issue as the same DR occurs twice
					{7, 500, 1},
					{-1, 0, 0},
					{-1, 0, 0},
				},
				SX1301Conf: []SX1301Conf{
					{
						Radio0: SX1301ConfRadio{
							Enable: true,
							Freq:   902700000,
						},
						Radio1: SX1301ConfRadio{
							Enable: true,
							Freq:   903700000,
						},
						ChanLoRaStd: SX1301ConfChanLoRaStd{
							Enable:          true,
							Radio:           0,
							IF:              300000,
							Bandwidth:       500000,
							SpreadingFactor: 8,
						},
						ChanMultiSF0: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     -400000,
						},
						ChanMultiSF1: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     -200000,
						},
						ChanMultiSF2: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     0,
						},
						ChanMultiSF3: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     200000,
						},
						ChanMultiSF4: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  0,
							IF:     400000,
						},
						ChanMultiSF5: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  1,
							IF:     -400000,
						},
						ChanMultiSF6: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  1,
							IF:     -200000,
						},
						ChanMultiSF7: SX1301ConfChanMultiSF{
							Enable: true,
							Radio:  1,
							IF:     0,
						},
					},
				},
			},
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)

			var conf config.Config
			conf.Backend.BasicStation.Concentrators = tst.Concentrators

			rc, err := GetRouterConfig(tst.Region, tst.NetIDs, tst.JoinEUIs, tst.FrequencyMin, tst.FrequencyMax, conf.Backend.BasicStation.Concentrators)
			assert.Equal(tst.ExpectedError, err)
			if err != nil {
				return
			}
			assert.Equal(tst.ExpectedRouterConfig, rc)
		})
	}
}
