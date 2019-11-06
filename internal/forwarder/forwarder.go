package forwarder

import (
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/metadata"
	"github.com/brocaar/chirpstack-api/go/gw"
	"github.com/brocaar/lorawan"
)

var alwaysSubscribe []lorawan.EUI64

func Setup(conf config.Config) error {
	b := backend.GetBackend()
	i := integration.GetIntegration()

	if b == nil {
		return errors.New("backend is not set")
	}

	if i == nil {
		return errors.New("integration is not set")
	}

	for _, c := range conf.Backend.SemtechUDP.Configuration {
		var gatewayID lorawan.EUI64
		if err := gatewayID.UnmarshalText([]byte(c.GatewayID)); err != nil {
			return errors.Wrap(err, "unmarshal gateway_id error")
		}

		if err := i.SubscribeGateway(gatewayID); err != nil {
			return errors.Wrap(err, "subscribe gateway error")
		}

		alwaysSubscribe = append(alwaysSubscribe, gatewayID)
	}

	go onConnectedLoop()
	go onDisconnectedLoop()

	go forwardUplinkFrameLoop()
	go forwardGatewayStatsLoop()
	go forwardDownlinkTxAckLoop()
	go forwardDownlinkFrameLoop()
	go forwardGatewayConfigurationLoop()

	return nil
}

func onConnectedLoop() {
	for gatewayID := range backend.GetBackend().GetConnectChan() {
		var found bool
		for _, gwID := range alwaysSubscribe {
			if gatewayID == gwID {
				found = true
			}
		}
		if found {
			break
		}

		if err := integration.GetIntegration().SubscribeGateway(gatewayID); err != nil {
			log.WithError(err).Error("subscribe gateway error")
		}
	}
}

func onDisconnectedLoop() {
	for gatewayID := range backend.GetBackend().GetDisconnectChan() {
		var found bool
		for _, gwID := range alwaysSubscribe {
			if gatewayID == gwID {
				found = true
			}
		}
		if found {
			break
		}

		if err := integration.GetIntegration().UnsubscribeGateway(gatewayID); err != nil {
			log.WithError(err).Error("unsubscribe gateway error")
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
			stats.MetaData = metadata.Get()

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
