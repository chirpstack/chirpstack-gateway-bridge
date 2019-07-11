package semtechudp

import (
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
