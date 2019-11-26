package mqtt

import (
	"os"
	"testing"
	"time"

	"github.com/brocaar/chirpstack-api/go/gw"
	"github.com/gofrs/uuid"

	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
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

	var conf config.Config
	conf.Integration.Marshaler = "json"
	conf.Integration.MQTT.EventTopicTemplate = "gateway/{{ .GatewayID }}/event/{{ .EventType }}"
	conf.Integration.MQTT.CommandTopicTemplate = "gateway/{{ .GatewayID }}/command/#"
	conf.Integration.MQTT.Auth.Type = "generic"
	conf.Integration.MQTT.Auth.Generic.Servers = []string{server}
	conf.Integration.MQTT.Auth.Generic.Username = username
	conf.Integration.MQTT.Auth.Generic.Password = password
	conf.Integration.MQTT.Auth.Generic.CleanSession = true

	var err error
	ts.backend, err = NewBackend(conf)
	assert.NoError(err)
	assert.NoError(ts.backend.SubscribeGateway(ts.gatewayID))
	time.Sleep(100 * time.Millisecond)
}

func (ts *MQTTBackendTestSuite) TearDownSuite() {
	ts.mqttClient.Disconnect(0)
	ts.backend.Close()
}

func (ts *MQTTBackendTestSuite) TestSubscribeGateway() {
	assert := require.New(ts.T())

	gatewayID := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}

	assert.NoError(ts.backend.SubscribeGateway(gatewayID))
	_, ok := ts.backend.gateways[gatewayID]
	assert.True(ok)

	ts.T().Run("Unsubscribe", func(t *testing.T) {
		assert := require.New(t)

		assert.NoError(ts.backend.UnsubscribeGateway(gatewayID))
		_, ok := ts.backend.gateways[gatewayID]
		assert.False(ok)
	})
}

func (ts *MQTTBackendTestSuite) TestPublishUplinkFrame() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	uplink := gw.UplinkFrame{
		PhyPayload: []byte{1, 2, 3, 4},
		RxInfo: &gw.UplinkRXInfo{
			UplinkId: id[:],
		},
	}

	uplinkFrameChan := make(chan gw.UplinkFrame)
	token := ts.mqttClient.Subscribe("gateway/+/event/up", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.UplinkFrame
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		uplinkFrameChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishEvent(ts.gatewayID, "up", id, &uplink))
	uplinkReceived := <-uplinkFrameChan
	assert.Equal(uplink, uplinkReceived)
}

func (ts *MQTTBackendTestSuite) TestGatewayStats() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	stats := gw.GatewayStats{
		GatewayId: ts.gatewayID[:],
		StatsId:   id[:],
	}

	statsChan := make(chan gw.GatewayStats)
	token := ts.mqttClient.Subscribe("gateway/+/event/stats", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.GatewayStats
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		statsChan <- stats
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishEvent(ts.gatewayID, "stats", id, &stats))
	statsReceived := <-statsChan
	assert.Equal(stats, statsReceived)
}

func (ts *MQTTBackendTestSuite) TestPublishDownlinkTXAck() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	txAck := gw.DownlinkTXAck{
		GatewayId:  ts.gatewayID[:],
		Token:      1234,
		DownlinkId: id[:],
	}

	txAckChan := make(chan gw.DownlinkTXAck)
	token := ts.mqttClient.Subscribe("gateway/+/event/ack", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.DownlinkTXAck
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		txAckChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishEvent(ts.gatewayID, "ack", id, &txAck))
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

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/down", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedDownlink := <-ts.backend.GetDownlinkFrameChan()
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

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/config", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedConfig := <-ts.backend.GetGatewayConfigurationChan()
	assert.Equal(config, receivedConfig)
}

func (ts *MQTTBackendTestSuite) TestGatewayCommandExecRequest() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	execReq := gw.GatewayCommandExecRequest{
		GatewayId: ts.gatewayID[:],
		ExecId:    id[:],
		Command:   "reboot",
		Environment: map[string]string{
			"FOO": "bar",
		},
	}

	b, err := ts.backend.marshal(&execReq)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/exec", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedExecReq := <-ts.backend.GetGatewayCommandExecRequestChan()
	assert.Equal(execReq, receivedExecReq)
}

func TestMQTTBackend(t *testing.T) {
	suite.Run(t, new(MQTTBackendTestSuite))
}
