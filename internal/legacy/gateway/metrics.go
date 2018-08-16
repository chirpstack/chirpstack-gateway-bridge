package gateway

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
)

var (
	gatewayEventCounter      func(string)
	gatewayWriteUDPTimer     func(string, func() error) error
	gatewayHandleTimer       func(string, func() error) error
	gatewayConfigHandleTimer func(func() error) error
)

func init() {
	ec := metrics.MustRegisterNewCounter(
		"gateway_event",
		"Per event type counter.",
		[]string{"event"},
	)

	wt := metrics.MustRegisterNewTimerWithError(
		"gateway_udp_write",
		"Per message-type UDP write duration tracking.",
		[]string{"type"},
	)

	ht := metrics.MustRegisterNewTimerWithError(
		"gateway_udp_received_handle",
		"Per message-type received UDP handling duration tracking.",
		[]string{"type"},
	)

	ch := metrics.MustRegisterNewTimerWithError(
		"gateway_config_handle",
		"Tracks the duration of configuration handling.",
		[]string{},
	)

	gatewayEventCounter = func(event string) {
		ec(prometheus.Labels{"event": event})
	}

	gatewayWriteUDPTimer = func(mType string, f func() error) error {
		return wt(prometheus.Labels{"type": mType}, f)
	}

	gatewayHandleTimer = func(mType string, f func() error) error {
		return ht(prometheus.Labels{"type": mType}, f)
	}

	gatewayConfigHandleTimer = func(f func() error) error {
		return ch(prometheus.Labels{}, f)
	}
}
