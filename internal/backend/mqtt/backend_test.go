package mqtt

import (
	"os"
	"testing"

	"github.com/brocaar/loraserver/api/gw"

	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/brocaar/lora-gateway-bridge/internal/backend/mqtt/auth"
	"github.com/brocaar/lorawan"
)

type MQTTBackendTestSuite struct {
	suite.Suite

	mqttClient paho.Client
	backend    *Backend
	gatewayID  lorawan.EUI64
}

func (ts *MQTTBackendTestSuite) SetupSuite() {
	assert := require.New(ts.T())

	log.SetLevel(log.ErrorLevel)

	server := "tcp://127.0.0.1:1883/1"
	var username string
	var password string

	if v := os.Getenv("TEST_MQTT_SERVER"); v != "" {
		server = v
	}
	if v := os.Getenv("TEST_MQTT_USERNAME"); v != "" {
		username = v
	}
	if v := os.Getenv("TEST_MQTT_PASSWORD"); v != "" {
		password = v
	}

	opts := paho.NewClientOptions().AddBroker(server).SetUsername(username).SetPassword(password)
	ts.mqttClient = paho.NewClient(opts)
	token := ts.mqttClient.Connect()
	token.Wait()
	assert.NoError(token.Error())

	ts.gatewayID = lorawan.EUI64{8, 7, 6, 5, 4, 3, 2, 1}

	var err error
	ts.backend, err = NewBackend(BackendConfig{
		UplinkTopicTemplate:   "gateway/{{ .MAC }}/rx",
		DownlinkTopicTemplate: "gateway/{{ .MAC }}/tx",
		StatsTopicTemplate:    "gateway/{{ .MAC }}/stats",
		AckTopicTemplate:      "gateway/{{ .MAC }}/ack",
		ConfigTopicTemplate:   "gateway/{{ .MAC }}/config",
		Marshaler:             "json",
		Auth: BackendAuthConfig{
			Type: "generic",
			Generic: auth.GenericConfig{
				Server:       server,
				Username:     username,
				Password:     password,
				CleanSession: true,
			},
		},
		AlwaysSubscribeMACs: []lorawan.EUI64{ts.gatewayID},
	})
	assert.NoError(err)
}

func (ts *MQTTBackendTestSuite) TearDownSuite() {
	ts.mqttClient.Disconnect(0)
	ts.backend.Close()
}

func (ts *MQTTBackendTestSuite) TestSubscribeGateway() {
	assert := require.New(ts.T())

	gatewayID := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}

	assert.NoError(ts.backend.SubscribeGateway(gatewayID))
	bl, ok := ts.backend.gateways[gatewayID]
	assert.True(ok)
	assert.False(bl)

	ts.T().Run("Unsubscribe", func(t *testing.T) {
		assert := require.New(t)

		assert.NoError(ts.backend.UnsubscribeGateway(gatewayID))
		_, ok := ts.backend.gateways[gatewayID]
		assert.False(ok)
	})
}

func (ts *MQTTBackendTestSuite) TestPublishUplinkFrame() {
	assert := require.New(ts.T())

	uplink := gw.UplinkFrame{
		PhyPayload: []byte{1, 2, 3, 4},
	}

	uplinkFrameChan := make(chan gw.UplinkFrame)
	token := ts.mqttClient.Subscribe("gateway/+/rx", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.UplinkFrame
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		uplinkFrameChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishUplinkFrame(ts.gatewayID, uplink))
	uplinkReceived := <-uplinkFrameChan
	assert.Equal(uplink, uplinkReceived)
}

func (ts *MQTTBackendTestSuite) TestGatewayStats() {
	assert := require.New(ts.T())

	stats := gw.GatewayStats{
		GatewayId: ts.gatewayID[:],
	}

	statsChan := make(chan gw.GatewayStats)
	token := ts.mqttClient.Subscribe("gateway/+/stats", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.GatewayStats
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		statsChan <- stats
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishGatewayStats(ts.gatewayID, stats))
	statsReceived := <-statsChan
	assert.Equal(stats, statsReceived)
}

func (ts *MQTTBackendTestSuite) TestPublishDownlinkTXAck() {
	assert := require.New(ts.T())

	txAck := gw.DownlinkTXAck{
		GatewayId: ts.gatewayID[:],
		Token:     1234,
	}

	txAckChan := make(chan gw.DownlinkTXAck)
	token := ts.mqttClient.Subscribe("gateway/+/ack", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.DownlinkTXAck
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		txAckChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishDownlinkTXAck(ts.gatewayID, txAck))
	txAckReceived := <-txAckChan
	assert.Equal(txAck, txAckReceived)
}

func (ts *MQTTBackendTestSuite) TestDownlinkFrameHandler() {
	assert := require.New(ts.T())

	downlink := gw.DownlinkFrame{
		PhyPayload: []byte{1, 2, 3, 4},
	}

	b, err := ts.backend.marshal(&downlink)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/tx", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedDownlink := <-ts.backend.DownlinkFrameChan()
	assert.Equal(downlink, receivedDownlink)
}

func (ts *MQTTBackendTestSuite) TestGatewayConfigHandler() {
	assert := require.New(ts.T())

	config := gw.GatewayConfiguration{
		GatewayId: ts.gatewayID[:],
		Version:   "123",
		Channels:  []*gw.ChannelConfiguration{},
	}

	b, err := ts.backend.marshal(&config)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/config", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedConfig := <-ts.backend.GatewayConfigurationChan()
	assert.Equal(config, receivedConfig)
}

func TestMQTTBackend(t *testing.T) {
	suite.Run(t, new(MQTTBackendTestSuite))
}
