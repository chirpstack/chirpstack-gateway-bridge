package basicstation

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
)

var (
	bsEventCounter             func(string)
	bsWebsocketSendCounter     func(string)
	bsWebsocketReceiveCounter  func(string)
	bsWebsocketPingPongCounter func(string)
)

func init() {
	ec := metrics.MustRegisterNewCounter(
		"backend_basicstation_event",
		"Per gateway event type counter.",
		[]string{"event"},
	)

	wsc := metrics.MustRegisterNewCounter(
		"backend_basicstation_websocket_send",
		"Per message-type websocket write counter.",
		[]string{"msgtype"},
	)

	wrc := metrics.MustRegisterNewCounter(
		"backend_basicstation_websocket_receive",
		"Per message-type websocket receive counter.",
		[]string{"msgtype"},
	)

	ppc := metrics.MustRegisterNewCounter(
		"backend_basicstation_websocket_ping_pong",
		"Websocket Ping/Pong counter.",
		[]string{"type"},
	)

	bsEventCounter = func(event string) {
		ec(prometheus.Labels{"event": event})
	}

	bsWebsocketReceiveCounter = func(msgtype string) {
		wsc(prometheus.Labels{"msgtype": msgtype})
	}

	bsWebsocketSendCounter = func(msgtype string) {
		wrc(prometheus.Labels{"msgtype": msgtype})
	}

	bsWebsocketPingPongCounter = func(typ string) {
		ppc(prometheus.Labels{"type": typ})
	}
}
