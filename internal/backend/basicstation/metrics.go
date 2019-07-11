package basicstation

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ppc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_basicstation_websocket_ping_pong_count",
		Help: "The number of WebSocket Ping/Pong requests sent and received (per event type).",
	}, []string{"type"})

	wsr = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_basicstation_websocket_received_count",
		Help: "The number of WebSocket messages received by the backend (per msgtype).",
	}, []string{"msgtype"})

	wss = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_basicstation_websocket_sent_count",
		Help: "The number of WebSocket messages sent by the backend (per msgtype).",
	}, []string{"msgtype"})

	gwc = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "backend_basicstation_gateway_connect_count",
		Help: "The number of gateway connections received by the backend.",
	})

	gwd = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "backend_basicstation_gateway_disconnect_count",
		Help: "The number of gateways that disconnected from the backend.",
	})
)

func websocketPingPongCounter(typ string) prometheus.Counter {
	return ppc.With(prometheus.Labels{"type": typ})
}

func websocketReceiveCounter(msgtype string) prometheus.Counter {
	return wsr.With(prometheus.Labels{"msgtype": msgtype})
}

func websocketSendCounter(msgtype string) prometheus.Counter {
	return wss.With(prometheus.Labels{"msgtype": msgtype})
}

func connectCounter() prometheus.Counter {
	return gwc
}

func disconnectCounter() prometheus.Counter {
	return gwd
}
