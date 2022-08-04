package mqtt

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chirpstack/chirpstack/api/go/v4/gw"
	"google.golang.org/protobuf/proto"

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
	assert.Equal(`{"gatewayId":"0807060504030201"}`, strings.ReplaceAll(string(ts.backend.clientOpts.WillPayload), " ", ""))
	assert.True(ts.backend.clientOpts.WillRetained)
}

func (ts *MQTTBackendTestSuite) TestConnStateOnline() {
	assert := require.New(ts.T())

	connStateChan := make(chan *gw.ConnState)
	token := ts.mqttClient.Subscribe("gateway/0807060504030201/state/conn", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.ConnState
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		connStateChan <- &pl
	})
	token.Wait()
	assert.NoError(token.Error())

	pl := <-connStateChan

	assert.True(proto.Equal(&gw.ConnState{
		GatewayId: ts.gatewayID.String(),
		State:     gw.ConnState_ONLINE,
	}, pl))

	token = ts.mqttClient.Unsubscribe("gateway/0807060504030201/state/conn")
	token.Wait()
	assert.NoError(token.Error())
}

func (ts *MQTTBackendTestSuite) TestSubscribeGateway() {
	assert := require.New(ts.T())

	gatewayID := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}
	connStateChan := make(chan *gw.ConnState)

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
		connStateChan <- &pl
	})
	token.Wait()
	assert.NoError(token.Error())

	pl := <-connStateChan

	assert.True(proto.Equal(&gw.ConnState{
		GatewayId: gatewayID.String(),
		State:     gw.ConnState_ONLINE,
	}, pl))

	ts.T().Run("Unsubscribe", func(t *testing.T) {
		assert := require.New(t)

		assert.NoError(ts.backend.SetGatewaySubscription(false, gatewayID))
		_, ok := ts.backend.gateways[gatewayID]
		assert.False(ok)

		pl := <-connStateChan

		assert.True(proto.Equal(&gw.ConnState{
			GatewayId: gatewayID.String(),
			State:     gw.ConnState_OFFLINE,
		}, pl))
	})

	token = ts.mqttClient.Unsubscribe("gateway/0102030405060708/state/conn")
	token.Wait()
	assert.NoError(token.Error())
}

func (ts *MQTTBackendTestSuite) TestPublishUplinkFrame() {
	assert := require.New(ts.T())

	uplink := gw.UplinkFrame{
		PhyPayload: []byte{1, 2, 3, 4},
		RxInfo: &gw.UplinkRxInfo{
			UplinkId: 123,
		},
	}

	uplinkFrameChan := make(chan *gw.UplinkFrame)
	token := ts.mqttClient.Subscribe("gateway/+/event/up", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.UplinkFrame
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		uplinkFrameChan <- &pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishEvent(ts.gatewayID, "up", uplink.GetRxInfo().GetUplinkId(), &uplink))
	uplinkReceived := <-uplinkFrameChan
	assert.True(proto.Equal(&uplink, uplinkReceived))
}

func (ts *MQTTBackendTestSuite) TestGatewayStats() {
	assert := require.New(ts.T())

	stats := gw.GatewayStats{
		GatewayId: ts.gatewayID.String(),
	}

	statsChan := make(chan *gw.GatewayStats)
	token := ts.mqttClient.Subscribe("gateway/+/event/stats", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.GatewayStats
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		statsChan <- &stats
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishEvent(ts.gatewayID, "stats", 0, &stats))
	statsReceived := <-statsChan
	assert.True(proto.Equal(&stats, statsReceived))
}

func (ts *MQTTBackendTestSuite) TestPublishDownlinkTxAck() {
	assert := require.New(ts.T())

	txAck := gw.DownlinkTxAck{
		GatewayId:  ts.gatewayID.String(),
		DownlinkId: 1234,
		Items: []*gw.DownlinkTxAckItem{
			{
				Status: gw.TxAckStatus_OK,
			},
		},
	}

	txAckChan := make(chan *gw.DownlinkTxAck)
	token := ts.mqttClient.Subscribe("gateway/+/event/ack", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.DownlinkTxAck
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		txAckChan <- &pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.NoError(ts.backend.PublishEvent(ts.gatewayID, "ack", txAck.GetDownlinkId(), &txAck))

	txAckReceived := <-txAckChan
	assert.True(proto.Equal(&txAck, txAckReceived))
}

func (ts *MQTTBackendTestSuite) TestPublishConnState() {
	assert := require.New(ts.T())

	// We publish first
	state := gw.ConnState{
		GatewayId: ts.gatewayID.String(),
		State:     gw.ConnState_ONLINE,
	}
	assert.NoError(ts.backend.PublishState(ts.gatewayID, "conn", &state))

	// And then subscribe to test that the message has been retained
	stateChan := make(chan *gw.ConnState)
	token := ts.mqttClient.Subscribe("gateway/0807060504030201/state/conn", 0, func(c paho.Client, msg paho.Message) {
		var pl gw.ConnState
		assert.NoError(ts.backend.unmarshal(msg.Payload(), &pl))
		stateChan <- &pl
	})
	token.Wait()
	assert.NoError(token.Error())

	assert.True(proto.Equal(&state, <-stateChan))

	token = ts.mqttClient.Unsubscribe("gateway/0807060504030201/state/conn")
	token.Wait()
	assert.NoError(token.Error())
}

func (ts *MQTTBackendTestSuite) TestDownlinkFrameHandler() {
	assert := require.New(ts.T())
	downlinkFrameChan := make(chan *gw.DownlinkFrame, 1)
	ts.backend.SetDownlinkFrameFunc(func(pl *gw.DownlinkFrame) {
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
	assert.True(proto.Equal(&downlink, receivedDownlink))
}

func (ts *MQTTBackendTestSuite) TestGatewayConfigHandler() {
	assert := require.New(ts.T())
	gatewayConfigurationChan := make(chan *gw.GatewayConfiguration, 1)
	ts.backend.SetGatewayConfigurationFunc(func(pl *gw.GatewayConfiguration) {
		gatewayConfigurationChan <- pl
	})

	config := gw.GatewayConfiguration{
		GatewayId: ts.gatewayID.String(),
		Version:   "123",
	}

	b, err := ts.backend.marshal(&config)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/config", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	receivedConfig := <-gatewayConfigurationChan
	assert.True(proto.Equal(&config, receivedConfig))
}

func (ts *MQTTBackendTestSuite) TestGatewayCommandExecRequest() {
	assert := require.New(ts.T())
	gatewayComandExecRequestChan := make(chan *gw.GatewayCommandExecRequest, 1)
	ts.backend.SetGatewayCommandExecRequestFunc(func(pl *gw.GatewayCommandExecRequest) {
		gatewayComandExecRequestChan <- pl
	})

	execReq := gw.GatewayCommandExecRequest{
		GatewayId: ts.gatewayID.String(),
		ExecId:    123,
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
	assert.True(proto.Equal(&execReq, receivedExecReq))
}

func (ts *MQTTBackendTestSuite) TestRawPacketForwarderCommand() {
	assert := require.New(ts.T())
	rawPacketForwarderCommandChan := make(chan *gw.RawPacketForwarderCommand, 1)
	ts.backend.SetRawPacketForwarderCommandFunc(func(pl *gw.RawPacketForwarderCommand) {
		rawPacketForwarderCommandChan <- pl
	})

	pl := gw.RawPacketForwarderCommand{
		GatewayId: ts.gatewayID.String(),
		Payload:   []byte{0x01, 0x02, 0x03, 0x04},
	}

	b, err := ts.backend.marshal(&pl)
	assert.NoError(err)

	token := ts.mqttClient.Publish("gateway/0807060504030201/command/raw", 0, false, b)
	token.Wait()
	assert.NoError(token.Error())

	received := <-rawPacketForwarderCommandChan
	assert.True(proto.Equal(&pl, received))
}

func TestMQTTBackend(t *testing.T) {
	suite.Run(t, new(MQTTBackendTestSuite))
}
