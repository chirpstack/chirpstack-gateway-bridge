package mqtt

import (
	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	mqttPublishTimer      func(string, func() error) error
	mqttConnectTimer      func(func() error) error
	mqttSubscribeTimer    func(func() error) error
	mqttUnsubscribeTimer  func(func() error) error
	mqttCommandCounter    func(string)
	mqttConnectionCounter func(string)
)

func init() {
	pt := metrics.MustRegisterNewTimerWithError(
		"integration_mqtt_publish",
		"Per event-type publish to MQTT broker duration tracking.",
		[]string{"type"},
	)

	mc := metrics.MustRegisterNewTimerWithError(
		"integration_mqtt_connect",
		"Duration of connecting to the MQTT broker.",
		[]string{},
	)

	ms := metrics.MustRegisterNewTimerWithError(
		"integration_mqtt_subscribe",
		"Duration of subscribing to a MQTT topic.",
		[]string{},
	)

	mus := metrics.MustRegisterNewTimerWithError(
		"integration_mqtt_unsubscribe",
		"Duration of unsubscribing from a MQTT topic.",
		[]string{},
	)

	connC := metrics.MustRegisterNewCounter(
		"integration_mqtt_connection",
		"Connection event counter.",
		[]string{"event"},
	)

	cc := metrics.MustRegisterNewCounter(
		"integration_mqtt_command",
		"Received command counter.",
		[]string{"event"},
	)

	mqttPublishTimer = func(mType string, f func() error) error {
		return pt(prometheus.Labels{"type": mType}, f)
	}

	mqttConnectTimer = func(f func() error) error {
		return mc(prometheus.Labels{}, f)
	}

	mqttSubscribeTimer = func(f func() error) error {
		return ms(prometheus.Labels{}, f)
	}

	mqttUnsubscribeTimer = func(f func() error) error {
		return mus(prometheus.Labels{}, f)
	}

	mqttCommandCounter = func(event string) {
		cc(prometheus.Labels{"event": event})
	}

	mqttConnectionCounter = func(event string) {
		connC(prometheus.Labels{"event": event})
	}
}
