// Package sx1301v1 contains helpers for generating configuration for Semtech SX1301v1 gateways.
package sx1301v1

import (
	"fmt"
	"sort"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
)

// radioBandwidthPerChannelBandwidth defines the bandwidth that a single radio
// can cover per channel bandwidth
var radioBandwidthPerChannelBandwidth = map[uint32]uint32{
	500000: 1100000, // 500kHz channel
	250000: 1000000, // 250kHz channel
	125000: 925000,  // 125kHz channel
}

// defaultRadioBandwidth defines the radio bandwidth in case the channel
// bandwidth does not match any of the above values.
const defaultRadioBandwidth uint32 = 925000

// channelByMinRadioCenterFreqency implements sort.Interface for []*gw.ChannelConfiguration.
// The sorting is based on the center frequency of the radio when placing the
// channel exactly on the left side of the available radio bandwidth.
type channelByMinRadioCenterFrequency []*gw.ChannelConfiguration

func (c channelByMinRadioCenterFrequency) Len() int      { return len(c) }
func (c channelByMinRadioCenterFrequency) Swap(i, j int) { c[i], c[j] = c[j], c[i] }
func (c channelByMinRadioCenterFrequency) Less(i, j int) bool {
	return c.minRadioCenterFreq(i) < c.minRadioCenterFreq(j)
}
func (c channelByMinRadioCenterFrequency) minRadioCenterFreq(i int) uint32 {
	var channelBandwidth uint32

	switch c[i].Modulation {
	case common.Modulation_LORA:
		modInfo := c[i].GetLoraModulationConfig()
		if modInfo != nil {
			channelBandwidth = modInfo.Bandwidth * 1000
		}
	case common.Modulation_FSK:
		modInfo := c[i].GetFskModulationConfig()
		if modInfo != nil {
			channelBandwidth = modInfo.Bandwidth * 1000
		}
	}

	radioBandwidth, ok := radioBandwidthPerChannelBandwidth[channelBandwidth]
	if !ok {
		radioBandwidth = defaultRadioBandwidth
	}
	return c[i].Frequency - (channelBandwidth / 2) + (radioBandwidth / 2)
}

// GetRadioFrequencies returns the center-frequencies for the two radios.
func GetRadioFrequencies(channels []*gw.ChannelConfiguration) ([2]uint32, error) {
	var radios [2]uint32

	// make sure the channels are sorted by the minimum radio center frequency
	sort.Sort(channelByMinRadioCenterFrequency(channels))

	for _, c := range channels {
		var channelBandwidth uint32

		switch c.Modulation {
		case common.Modulation_LORA:
			modInfo := c.GetLoraModulationConfig()
			if modInfo != nil {
				channelBandwidth = modInfo.Bandwidth * 1000
			}
		case common.Modulation_FSK:
			modInfo := c.GetFskModulationConfig()
			if modInfo != nil {
				channelBandwidth = modInfo.Bandwidth * 1000
			}
		}

		channelMax := c.Frequency + (channelBandwidth / 2)
		radioBandwidth, ok := radioBandwidthPerChannelBandwidth[channelBandwidth]
		if !ok {
			radioBandwidth = defaultRadioBandwidth
		}
		minRadioCenterFreq := c.Frequency - (channelBandwidth / 2) + (radioBandwidth / 2)

		for i := range radios {
			// the radio is not defined yet, use it
			if radios[i] == 0 {
				radios[i] = minRadioCenterFreq
				break
			}

			// channel fits within bandwidth of radio
			if channelMax <= radios[i]+(radioBandwidth/2) {
				break
			}

			// the channel does not fit
			if i == len(radios)-1 {
				return radios, fmt.Errorf("channel %d does not fit in radio bandwidth", c.Frequency)
			}
		}
	}

	return radios, nil
}

// GetRadioForChannel returns the radio number to which the channel must be assigned.
func GetRadioForChannel(radios [2]uint32, c *gw.ChannelConfiguration) (int, error) {
	var channelBandwidth uint32

	switch c.Modulation {
	case common.Modulation_LORA:
		modInfo := c.GetLoraModulationConfig()
		if modInfo != nil {
			channelBandwidth = modInfo.Bandwidth * 1000
		}
	case common.Modulation_FSK:
		modInfo := c.GetFskModulationConfig()
		if modInfo != nil {
			channelBandwidth = modInfo.Bandwidth * 1000
		}
	}

	channelMin := c.Frequency - (channelBandwidth / 2)
	channelMax := c.Frequency + (channelBandwidth / 2)
	radioBandwidth, ok := radioBandwidthPerChannelBandwidth[channelBandwidth]
	if !ok {
		radioBandwidth = defaultRadioBandwidth
	}

	for i, f := range radios {
		if channelMin >= f-(radioBandwidth/2) && channelMax <= f+(radioBandwidth/2) {
			return i, nil
		}
	}

	return 0, fmt.Errorf("channel %d does not fit in radio bandwidth", c.Frequency)
}
