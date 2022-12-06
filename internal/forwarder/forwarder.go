package forwarder

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/metadata"
	"github.com/brocaar/lorawan"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

// Setup configures the forwarder.
func Setup(conf config.Config) error {
	b := backend.GetBackend()
	i := integration.GetIntegration()

	if b == nil {
		return errors.New("backend is not set")
	}

	if i == nil {
		return errors.New("integration is not set")
	}

	// setup backend callbacks
	b.SetSubscribeEventFunc(gatewaySubscribeFunc)
	b.SetUplinkFrameFunc(uplinkFrameFunc)
	b.SetGatewayStatsFunc(gatewayStatsFunc)
	b.SetDownlinkTxAckFunc(downlinkTxAckFunc)
	b.SetRawPacketForwarderEventFunc(rawPacketForwarderEventFunc)

	// setup integration callbacks
	i.SetDownlinkFrameFunc(downlinkFrameFunc)
	i.SetGatewayConfigurationFunc(gatewayConfigurationFunc)
	i.SetRawPacketForwarderCommandFunc(rawPacketForwarderCommandFunc)

	return nil
}

func gatewaySubscribeFunc(pl events.Subscribe) {
	go func(pl events.Subscribe) {
		if err := integration.GetIntegration().SetGatewaySubscription(pl.Subscribe, pl.GatewayID); err != nil {
			log.WithError(err).Error("set gateway subscription error")
		}
	}(pl)
}

func uplinkFrameFunc(pl *gw.UplinkFrame) {
	go func(pl *gw.UplinkFrame) {
		var gatewayID lorawan.EUI64
		if err := gatewayID.UnmarshalText([]byte(pl.GetRxInfo().GetGatewayId())); err != nil {
			log.WithError(err).Error("decode gateway id error")
			return
		}

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventUp, pl.GetRxInfo().GetUplinkId(), pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id": gatewayID,
				"event_type": integration.EventUp,
				"uplink_id":  pl.GetRxInfo().GetUplinkId(),
			}).Error("publish event error")
		}
	}(pl)
}

func gatewayStatsFunc(pl *gw.GatewayStats) {
	go func(pl *gw.GatewayStats) {
		var gatewayID lorawan.EUI64
		if err := gatewayID.UnmarshalText([]byte(pl.GetGatewayId())); err != nil {
			log.WithError(err).Error("decode gateway id error")
			return
		}

		// add meta-data to stats
		if pl.Metadata == nil {
			pl.Metadata = make(map[string]string)
		}
		for k, v := range metadata.Get() {
			pl.Metadata[k] = v
		}

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventStats, 0, pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id": gatewayID,
				"event_type": integration.EventStats,
			}).Error("publish event error")
		}
	}(pl)
}

func downlinkTxAckFunc(pl *gw.DownlinkTxAck) {
	go func(pl *gw.DownlinkTxAck) {
		var gatewayID lorawan.EUI64
		if err := gatewayID.UnmarshalText([]byte(pl.GetGatewayId())); err != nil {
			log.WithError(err).Error("decode gateway id error")
			return
		}

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventAck, pl.GetDownlinkId(), pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id":  gatewayID,
				"event_type":  integration.EventAck,
				"downlink_id": pl.GetDownlinkId(),
			}).Error("publish event error")
		}
	}(pl)
}

func rawPacketForwarderEventFunc(pl *gw.RawPacketForwarderEvent) {
	go func(pl *gw.RawPacketForwarderEvent) {
		var gatewayID lorawan.EUI64
		if err := gatewayID.UnmarshalText([]byte(pl.GetGatewayId())); err != nil {
			log.WithError(err).Error("decode gateway id error")
			return
		}

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventRaw, 0, pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id": gatewayID,
				"event_type": integration.EventRaw,
			}).Error("publish event error")
		}
	}(pl)
}

func downlinkFrameFunc(pl *gw.DownlinkFrame) {
	go func(pl *gw.DownlinkFrame) {
		if err := backend.GetBackend().SendDownlinkFrame(pl); err != nil {
			log.WithError(err).Error("send downlink frame error")
		}
	}(pl)
}

func gatewayConfigurationFunc(pl *gw.GatewayConfiguration) {
	go func(pl *gw.GatewayConfiguration) {
		if err := backend.GetBackend().ApplyConfiguration(pl); err != nil {
			log.WithError(err).Error("apply gateway-configuration error")
		}
	}(pl)
}

func rawPacketForwarderCommandFunc(pl *gw.RawPacketForwarderCommand) {
	go func(pl *gw.RawPacketForwarderCommand) {
		if err := backend.GetBackend().RawPacketForwarderCommand(pl); err != nil {
			log.WithError(err).Error("raw packet-forwarder command error")
		}
	}(pl)
}
