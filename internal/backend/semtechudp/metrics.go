package semtechudp

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
)

var (
	eventCounter    func(string)
	udpWriteCounter func(string)
	udpReadCounter  func(string)
)

func init() {
	ec := metrics.MustRegisterNewCounter(
		"backend_semtechudp_event",
		"Per gateway event type counter.",
		[]string{"event"},
	)

	uwc := metrics.MustRegisterNewCounter(
		"backend_semtechudp_udp_write",
		"UDP packets written by packet type.",
		[]string{"packet_type"},
	)

	urc := metrics.MustRegisterNewCounter(
		"backend_semtechudp_udp_read",
		"UDP packets read by packet type.",
		[]string{"packet_type"},
	)

	eventCounter = func(event string) {
		ec(prometheus.Labels{"event": event})
	}

	udpWriteCounter = func(pt string) {
		uwc(prometheus.Labels{"packet_type": pt})
	}

	udpReadCounter = func(pt string) {
		urc(prometheus.Labels{"packet_type": pt})
	}
}
