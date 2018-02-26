package mqttpubsub

import (
	"os"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.ErrorLevel)
}

type conf struct {
	Server   string
	Username string
	Password string
}

func getConfig() *conf {
	config.C.Backend.MQTT.DownlinkTopicTemplate = "gateway/{{ .MAC }}/tx"
	config.C.Backend.MQTT.UplinkTopicTemplate = "gateway/{{ .MAC }}/rx"
	config.C.Backend.MQTT.StatsTopicTemplate = "gateway/{{ .MAC }}/stats"
	config.C.Backend.MQTT.AckTopicTemplate = "gateway/{{ .MAC }}/ack"

	c := &conf{
		Server: "tcp://127.0.0.1:1883",
	}

	if v := os.Getenv("TEST_MQTT_SERVER"); v != "" {
		c.Server = v
	}

	if v := os.Getenv("TEST_MQTT_USERNAME"); v != "" {
		c.Username = v
	}

	if v := os.Getenv("TEST_MQTT_PASSWORD"); v != "" {
		c.Password = v
	}

	return c
}
