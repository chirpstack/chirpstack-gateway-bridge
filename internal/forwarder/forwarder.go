package forwarder

import (
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/metadata"
	"github.com/brocaar/lorawan"
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

func uplinkFrameFunc(pl gw.UplinkFrame) {
	go func(pl gw.UplinkFrame) {
		var gatewayID lorawan.EUI64
		var uplinkID uuid.UUID
		copy(gatewayID[:], pl.GetRxInfo().GatewayId)
		copy(uplinkID[:], pl.GetRxInfo().UplinkId)

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventUp, uplinkID, &pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id": gatewayID,
				"event_type": integration.EventUp,
				"uplink_id":  uplinkID,
			}).Error("publish event error")
		}
	}(pl)
}

func gatewayStatsFunc(pl gw.GatewayStats) {
	go func(pl gw.GatewayStats) {
		var gatewayID lorawan.EUI64
		var statsID uuid.UUID
		copy(gatewayID[:], pl.GatewayId)
		copy(statsID[:], pl.StatsId)

		// add meta-data to stats
		if pl.MetaData == nil {
			pl.MetaData = make(map[string]string)
		}
		for k, v := range metadata.Get() {
			pl.MetaData[k] = v
		}

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventStats, statsID, &pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id": gatewayID,
				"event_type": integration.EventStats,
				"stats_id":   statsID,
			}).Error("publish event error")
		}
	}(pl)
}

func downlinkTxAckFunc(pl gw.DownlinkTXAck) {
	go func(pl gw.DownlinkTXAck) {
		var gatewayID lorawan.EUI64
		var downID uuid.UUID
		copy(gatewayID[:], pl.GatewayId)
		copy(downID[:], pl.DownlinkId)

		// for backwards compatibility
		for _, err := range pl.Items {
			if err.Status == gw.TxAckStatus_OK {
				pl.Error = ""
				break
			}

			pl.Error = err.String()
		}

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventAck, downID, &pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id":  gatewayID,
				"event_type":  integration.EventAck,
				"downlink_id": downID,
			}).Error("publish event error")
		}
	}(pl)
}

func rawPacketForwarderEventFunc(pl gw.RawPacketForwarderEvent) {
	go func(pl gw.RawPacketForwarderEvent) {
		var gatewayID lorawan.EUI64
		var rawID uuid.UUID
		copy(gatewayID[:], pl.GatewayId)
		copy(rawID[:], pl.RawId)

		if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventRaw, rawID, &pl); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"gateway_id": gatewayID,
				"event_type": integration.EventRaw,
				"raw_id":     rawID,
			}).Error("publish event error")
		}
	}(pl)
}

func downlinkFrameFunc(pl gw.DownlinkFrame) {
	go func(pl gw.DownlinkFrame) {
		if err := backend.GetBackend().SendDownlinkFrame(pl); err != nil {
			log.WithError(err).Error("send downlink frame error")
		}
	}(pl)
}

func gatewayConfigurationFunc(pl gw.GatewayConfiguration) {
	go func(pl gw.GatewayConfiguration) {
		if err := backend.GetBackend().ApplyConfiguration(pl); err != nil {
			log.WithError(err).Error("apply gateway-configuration error")
		}
	}(pl)
}

func rawPacketForwarderCommandFunc(pl gw.RawPacketForwarderCommand) {
	go func(pl gw.RawPacketForwarderCommand) {
		if err := backend.GetBackend().RawPacketForwarderCommand(pl); err != nil {
			log.WithError(err).Error("raw packet-forwarder command error")
		}
	}(pl)
}
