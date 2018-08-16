package mqttpubsub

import (
	"fmt"

	"github.com/brocaar/lora-gateway-bridge/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	mqttPublishTimer     func(string, func() error) error
	mqttHandleTimer      func(string, func() error) error
	mqttConnectTimer     func(func() error) error
	mqttSubscribeTimer   func(bool, func() error) error
	mqttUnsubscribeTimer func(func() error) error
	mqttEventCounter     func(string)
)

func init() {
	pt := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_publish",
		"Per message-type publish to MQTT broker duration tracking.",
		[]string{"type"},
	)

	ht := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_handle",
		"Per message-type handle duration tracking (note 'handled' means it is internally added to the queue). This should be instantaneously, if not it indicatest that the queue is blocked.",
		[]string{"type"},
	)

	mc := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_connect",
		"Duration of connecting to the MQTT broker.",
		[]string{},
	)

	ms := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_subscribe",
		"Duration of subscribing to a MQTT topic.",
		[]string{"multi"},
	)

	mus := metrics.MustRegisterNewTimerWithError(
		"backend_mqtt_unsubscribe",
		"Duration of unsubscribing from a MQTT topic.",
		[]string{},
	)

	ec := metrics.MustRegisterNewCounter(
		"backend_mqtt_event",
		"Per event type counter.",
		[]string{"event"},
	)

	mqttPublishTimer = func(mType string, f func() error) error {
		return pt(prometheus.Labels{"type": mType}, f)
	}

	mqttHandleTimer = func(mType string, f func() error) error {
		return ht(prometheus.Labels{"type": mType}, f)
	}
	mqttConnectTimer = func(f func() error) error {
		return mc(prometheus.Labels{}, f)
	}

	mqttSubscribeTimer = func(multi bool, f func() error) error {
		return ms(prometheus.Labels{"multi": fmt.Sprintf("%t", multi)}, f)
	}

	mqttUnsubscribeTimer = func(f func() error) error {
		return mus(prometheus.Labels{}, f)
	}

	mqttEventCounter = func(event string) {
		ec(prometheus.Labels{"event": event})
	}
}
