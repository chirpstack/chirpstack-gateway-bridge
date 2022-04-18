package mqtt

import (
	"os"
	"testing"
	"time"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
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
	conf.Integration.MQTT.StateTopicTemplate = "gateway/{{ .GatewayID }}/state/{{ .StateType }}"
	conf.Integration.MQTT.CommandTopicTemplate = "gateway/{{ .GatewayID }}/command/#"
	conf.Integration.MQTT.StateRetained = true
	conf.Integration.MQTT.Auth.Type = "generic"
	conf.Integration.MQTT.Auth.Generic.Servers = []string{server}
	conf.Integration.MQTT.Auth.Generic.Username = username
	conf.Integration.MQTT.Auth.Generic.Password = password
	conf.Integration.MQTT.Auth.Generic.CleanSession = true
	conf.Integration.MQTT.Auth.Generic.ClientID = ts.gatewayID.String()
	conf.Integration.MQTT.MaxTokenWait = time.Second

	var err error
	ts.backend, err = NewBackend(conf)
	assert.NoError(err)
	assert.NoError(ts.backend.Start())

	// The subscribe loop runs every 100ms, we will wait twice the time to make
	// sure the subscription is set.
	time.Sleep(200 * time.Millisecond)
}

func (ts *MQTTBackendTestSuite) TearDownSuite() {
	ts.mqttClient.Disconnect(0)
	ts.backend.Stop()
}

func (ts *MQTTBackendTestSuite) TestLastWill() {
	assert := require.New(ts.T())

	assert.True(ts.backend.clientOpts.WillEnabled)
	assert.Equal("gateway/0807060504030201/state/conn", ts.backend.clientOpts.WillTopic)
	assert.Equal(`{"gatewayID":"CAcGBQQDAgE=","state":"OFFLINE"}`, string(ts.backend.clientOpts.WillPayload))
	assert.True(ts.backend.clientOpts.WillRetained)
}

func (ts *MQTTBackendTestSuite) TestConnStateOnline() {
	assert := require.New(ts.T())

	connStateChan := make(chan gw.ConnState)
	token := ts.mqttClient.Subscribe("gateway/0807060504030201/state/conn", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.ConnState
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		connStateChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.Equal(gw.ConnState{
		GatewayId: ts.gatewayID[:],
		State:     gw.ConnState_ONLINE,
	}, <-connStateChan)

	token = ts.mqttClient.Unsubscribe("gateway/0807060504030201/state/conn")
	token.Wait()
	assert.NoError(token.Error())
}

func (ts *MQTTBackendTestSuite) TestSubscribeGateway() {
	assert := require.New(ts.T())

	gatewayID := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}
	connStateChan := make(chan gw.ConnState)

	assert.NoError(ts.backend.SetGatewaySubscription(true, gatewayID))
	_, ok := ts.backend.gateways[gatewayID]
	assert.True(ok)

	// Wait 200ms to make sure that the subscribe loop has picked up the
	// change and set the ConnState. If we subscribe too early, it is
	// possible that we get an (old) OFFLINE retained message.
	time.Sleep(200 * time.Millisecond)

	token := ts.mqttClient.Subscribe("gateway/0102030405060708/state/conn", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.ConnState
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		connStateChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.Equal(gw.ConnState{
		GatewayId: gatewayID[:],
		State:     gw.ConnState_ONLINE,
	}, <-connStateChan)

	ts.T().Run("Unsubscribe", func(t *testing.T) {
		assert := require.New(t)

		assert.NoError(ts.backend.SetGatewaySubscription(false, gatewayID))
		_, ok := ts.backend.gateways[gatewayID]
		assert.False(ok)

		assert.Equal(gw.ConnState{
			GatewayId: gatewayID[:],
			State:     gw.ConnState_OFFLINE,
		}, <-connStateChan)
	})

	token = ts.mqttClient.Unsubscribe("gateway/0102030405060708/state/conn")
	token.Wait()
	assert.NoError(token.Error())
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
		Items: []*gw.DownlinkTXAckItem{
			{
				Status: gw.TxAckStatus_OK,
			},
		},
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

func (ts *MQTTBackendTestSuite) TestPublishConnState() {
	assert := require.New(ts.T())

	// We publish first
	state := gw.ConnState{
		GatewayId: ts.gatewayID[:],
		State:     gw.ConnState_ONLINE,
	}
	assert.NoError(ts.backend.PublishState(ts.gatewayID, "conn", &state))

	// And then subscribe to test that the message has been retained
	stateChan := make(chan gw.ConnState)
	token := ts.mqttClient.Subscribe("gateway/0807060504030201/state/conn", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.ConnState
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		stateChan <- pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.Equal(state, <-stateChan)

	token = ts.mqttClient.Unsubscribe("gateway/0807060504030201/state/conn")
	token.Wait()
	assert.NoError(token.Error())
}

func (ts *MQTTBackendTestSuite) TestDownlinkFrameHandler() {
	assert := require.New(ts.T())
	downlinkFrameChan := make(chan gw.DownlinkFrame, 1)
	ts.backend.SetDownlinkFrameFunc(func(pl gw.DownlinkFrame) {
		downlinkFrameChan <- pl
	})

	downlink := gw.DownlinkFrame{
		Items: []*gw.DownlinkFrameItem{
			{
				PhyPayload: []byte{1, 2, 3, 4},
			},
		},
	}

	b, err := ts.backend.marshal(&downlink)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/down", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedDownlink := <-downlinkFrameChan
	assert.Equal(downlink, receivedDownlink)
}

func (ts *MQTTBackendTestSuite) TestGatewayConfigHandler() {
	assert := require.New(ts.T())
	gatewayConfigurationChan := make(chan gw.GatewayConfiguration, 1)
	ts.backend.SetGatewayConfigurationFunc(func(pl gw.GatewayConfiguration) {
		gatewayConfigurationChan <- pl
	})

	config := gw.GatewayConfiguration{
		GatewayId: ts.gatewayID[:],
		Version:   "123",
	}

	b, err := ts.backend.marshal(&config)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/config", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedConfig := <-gatewayConfigurationChan
	assert.Equal(config, receivedConfig)
}

func (ts *MQTTBackendTestSuite) TestGatewayCommandExecRequest() {
	assert := require.New(ts.T())
	gatewayComandExecRequestChan := make(chan gw.GatewayCommandExecRequest, 1)
	ts.backend.SetGatewayCommandExecRequestFunc(func(pl gw.GatewayCommandExecRequest) {
		gatewayComandExecRequestChan <- pl
	})

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

	receivedExecReq := <-gatewayComandExecRequestChan
	assert.Equal(execReq, receivedExecReq)
}

func (ts *MQTTBackendTestSuite) TestRawPacketForwarderCommand() {
	assert := require.New(ts.T())
	rawPacketForwarderCommandChan := make(chan gw.RawPacketForwarderCommand, 1)
	ts.backend.SetRawPacketForwarderCommandFunc(func(pl gw.RawPacketForwarderCommand) {
		rawPacketForwarderCommandChan <- pl
	})

	id, err := uuid.NewV4()
	assert.NoError(err)

	pl := gw.RawPacketForwarderCommand{
		GatewayId: ts.gatewayID[:],
		RawId:     id[:],
		Payload:   []byte{0x01, 0x02, 0x03, 0x04},
	}

	b, err := ts.backend.marshal(&pl)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/raw", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	received := <-rawPacketForwarderCommandChan
	assert.Equal(pl, received)
}

func TestMQTTBackend(t *testing.T) {
	suite.Run(t, new(MQTTBackendTestSuite))
}
