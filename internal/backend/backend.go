package backend

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/basicstation"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/concentratord"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/semtechudp"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
)

var backend Backend

// Setup configures the backend.
func Setup(conf config.Config) error {
	var err error

	switch conf.Backend.Type {
	case "semtech_udp":
		backend, err = semtechudp.NewBackend(conf)
	case "basic_station":
		backend, err = basicstation.NewBackend(conf)
	case "concentratord":
		backend, err = concentratord.NewBackend(conf)
	default:
		return fmt.Errorf("unknown backend type: %s", conf.Backend.Type)
	}

	if err != nil {
		return errors.Wrap(err, "new backend error")
	}

	return nil
}

// GetBackend returns the backend.
func GetBackend() Backend {
	return backend
}

// Backend defines the interface that a backend must implement
type Backend interface {
	// Stop closes the backend.
	Stop() error

	// Start starts the backend.
	Start() error

	// SetDownlinkTxAckFunc sets the DownlinkTXAck handler func.
	SetDownlinkTxAckFunc(func(gw.DownlinkTXAck))

	// SetGatewayStatsFunc sets the GatewayStats handler func.
	SetGatewayStatsFunc(func(gw.GatewayStats))

	// SetUplinkFrameFunc sets the UplinkFrame handler func.
	SetUplinkFrameFunc(func(gw.UplinkFrame))

	// SetRawPacketForwarderEventFunc sets the RawPacketForwarderEvent handler func.
	SetRawPacketForwarderEventFunc(func(gw.RawPacketForwarderEvent))

	// SetSubscribeEventFunc sets the Subscribe handler func.
	SetSubscribeEventFunc(func(events.Subscribe))

	// SendDownlinkFrame sends the given downlink frame.
	SendDownlinkFrame(gw.DownlinkFrame) error

	// ApplyConfiguration applies the given configuration to the gateway.
	ApplyConfiguration(gw.GatewayConfiguration) error

	// RawPacketForwarderCommand sends the given raw command to the packet-forwarder.
	RawPacketForwarderCommand(gw.RawPacketForwarderCommand) error
}
