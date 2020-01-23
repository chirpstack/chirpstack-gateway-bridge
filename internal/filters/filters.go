package filters

import (
	"encoding/binary"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
)

var netIDs []lorawan.NetID
var joinEUIs [][2]lorawan.EUI64

// Setup configures the filters package.
func Setup(conf config.Config) error {
	for _, netIDStr := range conf.Filters.NetIDs {
		var netID lorawan.NetID
		if err := netID.UnmarshalText([]byte(netIDStr)); err != nil {
			return errors.Wrap(err, "unmarshal NetID error")
		}

		netIDs = append(netIDs, netID)
		log.WithFields(log.Fields{
			"net_id": netID,
		}).Info("filters: NetID filter configured")
	}

	for _, set := range conf.Filters.JoinEUIs {
		var joinEUISet [2]lorawan.EUI64

		for i, s := range set {
			var joinEUI lorawan.EUI64
			if err := joinEUI.UnmarshalText([]byte(s)); err != nil {
				return errors.Wrap(err, "unmarshal JoinEUI error")
			}

			joinEUISet[i] = joinEUI
		}

		joinEUIs = append(joinEUIs, joinEUISet)

		log.WithFields(log.Fields{
			"join_eui_from": joinEUISet[0],
			"join_eui_to":   joinEUISet[1],
		}).Info("filters: JoinEUI range configured")
	}

	return nil
}

// MatchFilters will match the given LoRaWAN frame against the configured
// filters. This function returns true in the following cases:
// * If the PHYPayload matches the configured filters
// * If no filters are configured
// * In case the PHYPayload is not a valid LoRaWAN frame
func MatchFilters(b []byte) bool {
	// return true when no filters are configured
	if len(netIDs) == 0 && len(joinEUIs) == 0 {
		return true
	}

	// return true when we can't decode the LoRaWAN frame
	var phy lorawan.PHYPayload
	if err := phy.UnmarshalBinary(b); err != nil {
		log.WithError(err).Error("filters: unmarshal phypayload error")
		return true
	}

	switch phy.MHDR.MType {
	case lorawan.UnconfirmedDataUp, lorawan.ConfirmedDataUp:
		return filterDevAddr(phy)
	case lorawan.JoinRequest:
		return filterJoinRequest(phy)
	case lorawan.RejoinRequest:
		return filterRejoinRequest(phy)
	default:
		return true
	}
}

func matchNetIDFilter(netID lorawan.NetID) bool {
	if len(netIDs) == 0 {
		return true
	}

	for _, n := range netIDs {
		if n == netID {
			return true
		}
	}

	return false
}

func matchNetIDFilterForDevAddr(devAddr lorawan.DevAddr) bool {
	if len(netIDs) == 0 {
		return true
	}

	for _, netID := range netIDs {
		if devAddr.IsNetID(netID) {
			return true
		}
	}

	return false
}

func matchJoinEUIFilter(joinEUI lorawan.EUI64) bool {
	if len(joinEUIs) == 0 {
		return true
	}

	joinEUIInt := binary.BigEndian.Uint64(joinEUI[:])

	for _, pair := range joinEUIs {
		min := binary.BigEndian.Uint64(pair[0][:])
		max := binary.BigEndian.Uint64(pair[1][:])

		if joinEUIInt >= min && joinEUIInt <= max {
			return true
		}
	}

	return false
}

func filterDevAddr(phy lorawan.PHYPayload) bool {
	mac, ok := phy.MACPayload.(*lorawan.MACPayload)
	if !ok {
		return true
	}

	return matchNetIDFilterForDevAddr(mac.FHDR.DevAddr)
}

func filterJoinRequest(phy lorawan.PHYPayload) bool {
	jr, ok := phy.MACPayload.(*lorawan.JoinRequestPayload)
	if !ok {
		return true
	}

	return matchJoinEUIFilter(jr.JoinEUI)
}

func filterRejoinRequest(phy lorawan.PHYPayload) bool {
	switch v := phy.MACPayload.(type) {
	case *lorawan.RejoinRequestType02Payload:
		return matchNetIDFilter(v.NetID)
	case *lorawan.RejoinRequestType1Payload:
		return matchJoinEUIFilter(v.JoinEUI)
	default:
		return true
	}
}
