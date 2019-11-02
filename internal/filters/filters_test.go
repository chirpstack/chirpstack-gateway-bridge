package filters

import (
	"testing"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/stretchr/testify/require"
)

func TestFilters(t *testing.T) {
	netID0 := lorawan.NetID{0x00, 0x00, 0x00}
	devAddr00 := lorawan.DevAddr{0x01, 0x01, 0x01, 0x01}
	devAddr00.SetAddrPrefix(netID0)

	netID1 := lorawan.NetID{0x00, 0x00, 0x01}
	devAddr10 := lorawan.DevAddr{0x01, 0x01, 0x01, 0x01}
	devAddr10.SetAddrPrefix(netID1)

	tests := []struct {
		Name           string
		NetIDFilters   []string
		JoinEUIFilters [][2]string
		PHYPayload     lorawan.PHYPayload
		Expected       bool
	}{
		{
			Name: "join-request, no filter",
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.JoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.JoinRequestPayload{
					JoinEUI: lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				},
			},
			Expected: true,
		},
		{
			Name: "join-request matching JoinEUI - 1",
			JoinEUIFilters: [][2]string{
				[2]string{"0000000000000001", "0000000000000002"},
			},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.JoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.JoinRequestPayload{
					JoinEUI: lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
				},
			},
			Expected: true,
		},
		{
			Name: "join-request matching JoinEUI - 2",
			JoinEUIFilters: [][2]string{
				[2]string{"0000000000000001", "0000000000000002"},
			},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.JoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.JoinRequestPayload{
					JoinEUI: lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
				},
			},
			Expected: true,
		},
		{
			Name: "join-request not matching JoinEUI - 1",
			JoinEUIFilters: [][2]string{
				[2]string{"0000000000000001", "0000000000000002"},
			},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.JoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.JoinRequestPayload{
					JoinEUI: lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				},
			},
			Expected: false,
		},
		{
			Name: "join-request not matching JoinEUI - 2",
			JoinEUIFilters: [][2]string{
				[2]string{"0000000000000001", "0000000000000002"},
			},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.JoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.JoinRequestPayload{
					JoinEUI: lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
				},
			},
			Expected: false,
		},
		{
			Name: "rejoin 1 not matching JoinEUI",
			JoinEUIFilters: [][2]string{
				[2]string{"0000000000000001", "0000000000000002"},
			},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.RejoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.RejoinRequestType1Payload{
					RejoinType: lorawan.RejoinRequestType1,
					JoinEUI:    lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03},
				},
			},
			Expected: false,
		},
		{
			Name: "rejoin 1 matching JoinEUI",
			JoinEUIFilters: [][2]string{
				[2]string{"0000000000000001", "0000000000000002"},
			},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.RejoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.RejoinRequestType1Payload{
					RejoinType: lorawan.RejoinRequestType1,
					JoinEUI:    lorawan.EUI64{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
				},
			},
			Expected: true,
		},
		{
			Name: "uplink data, no filter",
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.UnconfirmedDataUp,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.MACPayload{
					FHDR: lorawan.FHDR{
						DevAddr: devAddr00,
					},
				},
			},
			Expected: true,
		},
		{
			Name:         "uplink data NetID match",
			NetIDFilters: []string{netID0.String()},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.UnconfirmedDataUp,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.MACPayload{
					FHDR: lorawan.FHDR{
						DevAddr: devAddr00,
					},
				},
			},
			Expected: true,
		},
		{
			Name:         "uplink data NetID no match",
			NetIDFilters: []string{netID0.String()},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.UnconfirmedDataUp,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.MACPayload{
					FHDR: lorawan.FHDR{
						DevAddr: devAddr10,
					},
				},
			},
			Expected: false,
		},
		{
			Name:         "rejoin request 0/2 NetID match",
			NetIDFilters: []string{netID0.String()},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.RejoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.RejoinRequestType02Payload{
					RejoinType: lorawan.RejoinRequestType0,
					NetID:      netID0,
				},
			},
			Expected: true,
		},
		{
			Name:         "rejoin request 0/2 NetID match",
			NetIDFilters: []string{netID0.String()},
			PHYPayload: lorawan.PHYPayload{
				MHDR: lorawan.MHDR{
					MType: lorawan.RejoinRequest,
					Major: lorawan.LoRaWANR1,
				},
				MACPayload: &lorawan.RejoinRequestType02Payload{
					RejoinType: lorawan.RejoinRequestType0,
					NetID:      netID1,
				},
			},
			Expected: false,
		},
	}

	for _, tst := range tests {
		t.Run(tst.Name, func(t *testing.T) {
			assert := require.New(t)

			netIDs = nil
			joinEUIs = nil

			var conf config.Config
			conf.Filters.NetIDs = tst.NetIDFilters
			conf.Filters.JoinEUIs = tst.JoinEUIFilters

			assert.NoError(Setup(conf))

			b, err := tst.PHYPayload.MarshalBinary()
			assert.NoError(err)

			assert.Equal(tst.Expected, MatchFilters(b))
		})
	}
}
