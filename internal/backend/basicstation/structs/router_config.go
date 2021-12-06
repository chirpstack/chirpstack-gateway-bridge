package structs

import (
	"encoding/binary"
	"fmt"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config/sx1301v1"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/pkg/errors"
)

var regionNameMapping = map[band.Name]string{
	band.AS923: "AS923",
	band.AU915: "AU915",
	band.CN470: "CN470",
	band.CN779: "CN779",
	band.EU433: "EU433",
	band.EU868: "EU863",
	band.IN865: "IN865",
	band.KR920: "KR920",
	band.US915: "US902",
	band.RU864: "RU864",
}

// RouterConfig implements the router-config message.
type RouterConfig struct {
	MessageType MessageType  `json:"msgtype"`
	NetID       []uint32     `json:"NetID"`
	JoinEui     [][]uint64   `json:"JoinEui"`
	Region      string       `json:"region"`
	HWSpec      string       `json:"hwspec"`
	FreqRange   []uint32     `json:"freq_range"`
	DRs         [][]int      `json:"DRs"`
	SX1301Conf  []SX1301Conf `json:"sx1301_conf"`
}

// SX1301Conf implements a single SX1301 configuration.
type SX1301Conf struct {
	Radio0       SX1301ConfRadio       `json:"radio_0"`
	Radio1       SX1301ConfRadio       `json:"radio_1"`
	ChanFSK      SX1301ConfChanFSK     `json:"chan_FSK"`
	ChanLoRaStd  SX1301ConfChanLoRaStd `json:"chan_Lora_std"`
	ChanMultiSF0 SX1301ConfChanMultiSF `json:"chan_multiSF_0"`
	ChanMultiSF1 SX1301ConfChanMultiSF `json:"chan_multiSF_1"`
	ChanMultiSF2 SX1301ConfChanMultiSF `json:"chan_multiSF_2"`
	ChanMultiSF3 SX1301ConfChanMultiSF `json:"chan_multiSF_3"`
	ChanMultiSF4 SX1301ConfChanMultiSF `json:"chan_multiSF_4"`
	ChanMultiSF5 SX1301ConfChanMultiSF `json:"chan_multiSF_5"`
	ChanMultiSF6 SX1301ConfChanMultiSF `json:"chan_multiSF_6"`
	ChanMultiSF7 SX1301ConfChanMultiSF `json:"chan_multiSF_7"`
}

// SX1301ConfRadio implements a SX1301 radio configuration.
type SX1301ConfRadio struct {
	Enable bool   `json:"enable"`
	Freq   uint32 `json:"freq"`
}

// SX1301ConfChanFSK implements the FSK channel configuration.
type SX1301ConfChanFSK struct {
	Enable bool `json:"enable"`
}

// SX1301ConfChanLoRaStd implements the LoRa (single SF) configuration.
type SX1301ConfChanLoRaStd struct {
	Enable          bool   `json:"enable"`
	Radio           int    `json:"radio"`
	IF              int    `json:"if"`
	Bandwidth       uint32 `json:"bandwidth,omitempty"`
	SpreadingFactor uint32 `json:"spread_factor,omitempty"`
}

// SX1301ConfChanMultiSF implements the LoRa multi SF configuration.
type SX1301ConfChanMultiSF struct {
	Enable bool `json:"enable"`
	Radio  int  `json:"radio"`
	IF     int  `json:"if"`
}

// GetRouterConfig returns the router-config message.
func GetRouterConfig(region band.Name, netIDs []lorawan.NetID, joinEUIs [][2]lorawan.EUI64, freqMin, freqMax uint32, concentrators []config.BasicStationConcentrator) (RouterConfig, error) {
	concentratorCount := len(concentrators)

	c := RouterConfig{
		MessageType: RouterConfigMessage,
		Region:      regionNameMapping[region],
		HWSpec:      fmt.Sprintf("sx1301/%d", concentratorCount),
		FreqRange:   []uint32{freqMin, freqMax},
		SX1301Conf:  make([]SX1301Conf, concentratorCount),
	}

	// set NetID filter
	for _, netID := range netIDs {
		c.NetID = append(c.NetID, binary.BigEndian.Uint32(append([]byte{0x00}, netID[:]...)))
	}

	// Set JoinEUI filter
	for _, set := range joinEUIs {
		c.JoinEui = append(c.JoinEui, []uint64{
			binary.BigEndian.Uint64(set[0][:]),
			binary.BigEndian.Uint64(set[1][:]),
		})
	}

	// Set data-rates
	b, err := band.GetConfig(region, false, lorawan.DwellTimeNoLimit)
	if err != nil {
		return c, errors.Wrap(err, "get band config error")
	}
	for i := 0; i < 16; i++ {
		dr, err := b.GetDataRate(i)
		if err != nil || (dr.Modulation != band.LoRaModulation && dr.Modulation != band.FSKModulation) {
			c.DRs = append(c.DRs, []int{-1, 0, 0})
			continue
		}

		var dnOnly int
		if _, err := b.GetDataRateIndex(true, dr); err != nil {
			dnOnly = 1
		}

		c.DRs = append(c.DRs, []int{
			dr.SpreadFactor,
			dr.Bandwidth,
			dnOnly,
		})
	}

	// Iterate over concentrators
	for concentratorNum, concentratorConf := range concentrators {
		var channelConfigs []*gw.ChannelConfiguration

		for _, freq := range concentratorConf.MultiSF.Frequencies {
			channelConfigs = append(channelConfigs, &gw.ChannelConfiguration{
				Frequency:  freq,
				Modulation: common.Modulation_LORA,
				ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
					LoraModulationConfig: &gw.LoRaModulationConfig{
						Bandwidth:        125,
						SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
					},
				},
			})
		}

		if fskFreq := concentratorConf.FSK.Frequency; fskFreq != 0 {
			channelConfigs = append(channelConfigs, &gw.ChannelConfiguration{
				Frequency:  fskFreq,
				Modulation: common.Modulation_FSK,
				ModulationConfig: &gw.ChannelConfiguration_FskModulationConfig{
					FskModulationConfig: &gw.FSKModulationConfig{
						Bandwidth: 125,
						Bitrate:   50000,
					},
				},
			})
		}

		if loraSTDFreq := concentratorConf.LoRaSTD.Frequency; loraSTDFreq != 0 {
			channelConfigs = append(channelConfigs, &gw.ChannelConfiguration{
				Frequency:  loraSTDFreq,
				Modulation: common.Modulation_LORA,
				ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
					LoraModulationConfig: &gw.LoRaModulationConfig{
						Bandwidth:        concentratorConf.LoRaSTD.Bandwidth / 1000,
						SpreadingFactors: []uint32{concentratorConf.LoRaSTD.SpreadingFactor},
					},
				},
			})
		}

		// Get radio frequencies
		radioFrequencies, err := sx1301v1.GetRadioFrequencies(channelConfigs)
		if err != nil {
			return c, errors.Wrap(err, "get radio frequencies error")
		}

		// set radios
		for i, f := range radioFrequencies {
			switch i {
			case 0:
				c.SX1301Conf[concentratorNum].Radio0.Enable = f != 0
				c.SX1301Conf[concentratorNum].Radio0.Freq = f
			case 1:
				c.SX1301Conf[concentratorNum].Radio1.Enable = f != 0
				c.SX1301Conf[concentratorNum].Radio1.Freq = f
			}
		}

		// set channels
		var channelI int
		for _, channel := range channelConfigs {
			r, err := sx1301v1.GetRadioForChannel(radioFrequencies, channel)
			if err != nil {
				return c, errors.Wrap(err, "get radio for channel error")
			}

			switch channel.Modulation {
			case common.Modulation_LORA:
				modInfo := channel.GetLoraModulationConfig()
				if modInfo == nil {
					continue
				}

				if len(modInfo.SpreadingFactors) == 1 {
					c.SX1301Conf[concentratorNum].ChanLoRaStd = SX1301ConfChanLoRaStd{
						Enable:          true,
						Radio:           r,
						IF:              int(channel.Frequency) - int(radioFrequencies[r]),
						Bandwidth:       modInfo.Bandwidth * 1000,
						SpreadingFactor: modInfo.SpreadingFactors[0],
					}

				} else {
					multiFSChan := SX1301ConfChanMultiSF{
						Enable: true,
						Radio:  r,
						IF:     int(channel.Frequency) - int(radioFrequencies[r]),
					}

					switch channelI {
					case 0:
						c.SX1301Conf[concentratorNum].ChanMultiSF0 = multiFSChan
					case 1:
						c.SX1301Conf[concentratorNum].ChanMultiSF1 = multiFSChan
					case 2:
						c.SX1301Conf[concentratorNum].ChanMultiSF2 = multiFSChan
					case 3:
						c.SX1301Conf[concentratorNum].ChanMultiSF3 = multiFSChan
					case 4:
						c.SX1301Conf[concentratorNum].ChanMultiSF4 = multiFSChan
					case 5:
						c.SX1301Conf[concentratorNum].ChanMultiSF5 = multiFSChan
					case 6:
						c.SX1301Conf[concentratorNum].ChanMultiSF6 = multiFSChan
					case 7:
						c.SX1301Conf[concentratorNum].ChanMultiSF7 = multiFSChan
					}

					channelI++
				}
			case common.Modulation_FSK:
				c.SX1301Conf[concentratorNum].ChanFSK = SX1301ConfChanFSK{
					Enable: true,
				}
			}
		}
	}

	return c, nil
}
