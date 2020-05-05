package forwarder

import (
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend"
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

	go gatewaySubscribeLoop()
	go forwardUplinkFrameLoop()
	go forwardGatewayStatsLoop()
	go forwardDownlinkTxAckLoop()
	go forwardDownlinkFrameLoop()
	go forwardGatewayConfigurationLoop()
	go forwardRawPacketForwarderCommandLoop()
	go forwardRawPacketForwarderEventLoop()

	return nil
}

func gatewaySubscribeLoop() {
	for event := range backend.GetBackend().GetSubscribeEventChan() {
		if err := integration.GetIntegration().SetGatewaySubscription(event.Subscribe, event.GatewayID); err != nil {
			log.WithError(err).Error("set gateway subscription error")
		}
	}
}

func forwardUplinkFrameLoop() {
	for uplinkFrame := range backend.GetBackend().GetUplinkFrameChan() {
		go func(uplinkFrame gw.UplinkFrame) {
			var gatewayID lorawan.EUI64
			var uplinkID uuid.UUID
			copy(gatewayID[:], uplinkFrame.RxInfo.GatewayId)
			copy(uplinkID[:], uplinkFrame.RxInfo.UplinkId)

			if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventUp, uplinkID, &uplinkFrame); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"gateway_id": gatewayID,
					"event_type": integration.EventUp,
					"uplink_id":  uplinkID,
				}).Error("publish event error")
			}
		}(uplinkFrame)
	}
}

func forwardGatewayStatsLoop() {
	for stats := range backend.GetBackend().GetGatewayStatsChan() {
		go func(stats gw.GatewayStats) {
			var gatewayID lorawan.EUI64
			var statsID uuid.UUID
			copy(gatewayID[:], stats.GatewayId)
			copy(statsID[:], stats.StatsId)

			// add meta-data to stats
			if stats.MetaData == nil {
				stats.MetaData = make(map[string]string)
			}
			for k, v := range metadata.Get() {
				stats.MetaData[k] = v
			}

			if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventStats, statsID, &stats); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"gateway_id": gatewayID,
					"event_type": integration.EventStats,
					"stats_id":   statsID,
				}).Error("publish event error")
			}
		}(stats)
	}
}

func forwardDownlinkTxAckLoop() {
	for txAck := range backend.GetBackend().GetDownlinkTXAckChan() {
		go func(txAck gw.DownlinkTXAck) {
			var gatewayID lorawan.EUI64
			copy(gatewayID[:], txAck.GatewayId)

			var downID uuid.UUID
			copy(downID[:], txAck.DownlinkId)

			// for backwards compatibility
			for _, err := range txAck.Items {
				if err.Status == gw.TxAckStatus_OK {
					txAck.Error = ""
					break
				}

				txAck.Error = err.String()
			}

			if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventAck, downID, &txAck); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"gateway_id":  gatewayID,
					"event_type":  integration.EventAck,
					"downlink_id": downID,
				}).Error("publish event error")
			}
		}(txAck)
	}
}

func forwardRawPacketForwarderEventLoop() {
	for raw := range backend.GetBackend().GetRawPacketForwarderEventChan() {
		go func(raw gw.RawPacketForwarderEvent) {
			var gatewayID lorawan.EUI64
			copy(gatewayID[:], raw.GatewayId)

			var rawID uuid.UUID
			copy(rawID[:], raw.RawId)

			if err := integration.GetIntegration().PublishEvent(gatewayID, integration.EventRaw, rawID, &raw); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"gateway_id": gatewayID,
					"event_type": integration.EventRaw,
					"raw_id":     rawID,
				}).Error("publish event error")
			}
		}(raw)
	}
}

func forwardDownlinkFrameLoop() {
	for downlinkFrame := range integration.GetIntegration().GetDownlinkFrameChan() {
		go func(downlinkFrame gw.DownlinkFrame) {
			if err := backend.GetBackend().SendDownlinkFrame(downlinkFrame); err != nil {
				log.WithError(err).Error("send downlink frame error")
			}
		}(downlinkFrame)
	}
}

func forwardGatewayConfigurationLoop() {
	for gatewayConfig := range integration.GetIntegration().GetGatewayConfigurationChan() {
		go func(gatewayConfig gw.GatewayConfiguration) {
			if err := backend.GetBackend().ApplyConfiguration(gatewayConfig); err != nil {
				log.WithError(err).Error("apply gateway-configuration error")
			}
		}(gatewayConfig)
	}
}

func forwardRawPacketForwarderCommandLoop() {
	for raw := range integration.GetIntegration().GetRawPacketForwarderChan() {
		go func(raw gw.RawPacketForwarderCommand) {
			if err := backend.GetBackend().RawPacketForwarderCommand(raw); err != nil {
				log.WithError(err).Error("raw packet-forwarder command error")
			}
		}(raw)
	}
}
