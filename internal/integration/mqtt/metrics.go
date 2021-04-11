package mqtt

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "integration_mqtt_event_count",
		Help: "The number of gateway events published by the MQTT integration (per event).",
	}, []string{"event"})

	sc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "integration_mqtt_state_count",
		Help: "The number of gateway states published by the MQTT integration (per state).",
	}, []string{"state"})

	cc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "integration_mqtt_command_count",
		Help: "The number of commands received by the MQTT integration (per command).",
	}, []string{"command"})

	mqttc = promauto.NewCounter(prometheus.CounterOpts{
		Name: "integration_mqtt_connect_count",
		Help: "The number of times the integration connected to the MQTT broker.",
	})

	mqttd = promauto.NewCounter(prometheus.CounterOpts{
		Name: "integration_mqtt_disconnect_count",
		Help: "The number of times the integration disconnected from the MQTT broker.",
	})

	mqttr = promauto.NewCounter(prometheus.CounterOpts{
		Name: "integration_mqtt_reconnect_count",
		Help: "The number of times the integration reconnected to the MQTT broker (this also increments the disconnect and connect counters).",
	})
)

func mqttEventCounter(e string) prometheus.Counter {
	return pc.With(prometheus.Labels{"event": e})
}

func mqttStateCounter(s string) prometheus.Counter {
	return sc.With(prometheus.Labels{"state": s})
}

func mqttCommandCounter(c string) prometheus.Counter {
	return cc.With(prometheus.Labels{"command": c})
}

func mqttConnectCounter() prometheus.Counter {
	return mqttc
}

func mqttDisconnectCounter() prometheus.Counter {
	return mqttd
}

func mqttReconnectCounter() prometheus.Counter {
	return mqttr
}
