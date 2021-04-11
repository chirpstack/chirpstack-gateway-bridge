package auth

import (
	"testing"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/stretchr/testify/require"
)

func TestGenericAuthentication(t *testing.T) {
	gatewayID := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}

	var conf config.Config
	conf.Integration.Marshaler = "json"
	conf.Integration.MQTT.EventTopicTemplate = "gateway/{{ .GatewayID }}/event/{{ .EventType }}"
	conf.Integration.MQTT.StateTopicTemplate = "gateway/{{ .GatewayID }}/state/{{ .StateType }}"
	conf.Integration.MQTT.CommandTopicTemplate = "gateway/{{ .GatewayID }}/command/#"
	conf.Integration.MQTT.Auth.Type = "generic"
	conf.Integration.MQTT.Auth.Generic.Servers = []string{"tcp://localhost:1883"}
	conf.Integration.MQTT.Auth.Generic.Username = "foo"
	conf.Integration.MQTT.Auth.Generic.Password = "bar"
	conf.Integration.MQTT.Auth.Generic.CleanSession = true
	conf.Integration.MQTT.Auth.Generic.ClientID = gatewayID.String()

	t.Run("New", func(t *testing.T) {
		assert := require.New(t)

		auth, err := NewGenericAuthentication(conf)
		assert.NoError(err)

		t.Run("GetGatewayID", func(t *testing.T) {
			assert := require.New(t)
			assert.Equal(&gatewayID, auth.GetGatewayID())
		})
	})
}
