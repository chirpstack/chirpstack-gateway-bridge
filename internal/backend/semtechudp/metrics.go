package semtechudp

import (
	"github.com/brocaar/lorawan"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	uwc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_semtechudp_udp_sent_count",
		Help: "The number of UDP packets sent by the backend (per packet_type).",
	}, []string{"packet_type"})

	urc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_semtechudp_udp_received_count",
		Help: "The number of UDP packets received by the backend (per packet_type).",
	}, []string{"packet_type"})

	gwc = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backend_semtechudp_gateway_connect_count",
		Help: "The number of gateway connections received by the backend.",
	})

	gwd = promauto.NewCounter(prometheus.CounterOpts{
		Name: "backend_semtechudp_gateway_diconnect_count",
		Help: "The number of gateways that disconnected from the backend.",
	})

	ackr = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "backend_semtechdup_gateway_ack_rate",
		Help: "The percentage of upstream datagrams that were acknowledged.",
	}, []string{"gateway_id"})

	ackrc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_semtechudp_gateway_ack_rate_count",
		Help: "The number of ack-rates reported.",
	}, []string{"gateway_id"})
)

func udpWriteCounter(pt string) prometheus.Counter {
	return uwc.With(prometheus.Labels{"packet_type": pt})
}

func udpReadCounter(pt string) prometheus.Counter {
	return urc.With(prometheus.Labels{"packet_type": pt})
}

func connectCounter() prometheus.Counter {
	return gwc
}

func disconnectCounter() prometheus.Counter {
	return gwd
}

func ackRate(gatewayID lorawan.EUI64) prometheus.Gauge {
	return ackr.With(prometheus.Labels{"gateway_id": gatewayID.String()})
}

func ackRateCounter(gatewayID lorawan.EUI64) prometheus.Counter {
	return ackrc.With(prometheus.Labels{"gateway_id": gatewayID.String()})
}
