package concentratord

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"

	"github.com/go-zeromq/zmq4"
	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
)

type BackendTestSuite struct {
	suite.Suite

	backend *Backend
	pubSock zmq4.Socket
	repSock zmq4.Socket
}

func (ts *BackendTestSuite) SetupSuite() {
	log.SetLevel(log.ErrorLevel)
}

func (ts *BackendTestSuite) SetupTest() {
	assert := require.New(ts.T())

	tempDir, err := ioutil.TempDir("", "test")
	assert.NoError(err)

	ts.pubSock = zmq4.NewPub(context.Background())
	ts.repSock = zmq4.NewRep(context.Background())

	assert.NoError(ts.pubSock.Listen(fmt.Sprintf("ipc://%s/events", tempDir)))
	assert.NoError(ts.repSock.Listen(fmt.Sprintf("ipc://%s/commands", tempDir)))

	var conf config.Config
	conf.Backend.Concentratord.EventURL = fmt.Sprintf("ipc://%s/events", tempDir)
	conf.Backend.Concentratord.CommandURL = fmt.Sprintf("ipc://%s/commands", tempDir)
	conf.Backend.Concentratord.CRCCheck = true

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// NewBackend expects the Gateway ID
		msg, err := ts.repSock.Recv()
		assert.NoError(err)
		assert.Equal("gateway_id", string(msg.Bytes()))
		assert.NoError(ts.repSock.Send(zmq4.NewMsg([]byte{1, 2, 3, 4, 5, 6, 7, 8})))
		wg.Done()
	}()

	ts.backend, err = NewBackend(conf)
	assert.NoError(err)

	subscribeEventChan := make(chan events.Subscribe, 1)
	ts.backend.subscribeEventFunc = func(pl events.Subscribe) {
		subscribeEventChan <- pl
	}

	assert.NoError(ts.backend.Start())
	wg.Wait()

	assert.Equal(events.Subscribe{

		Subscribe: true,
		GatewayID: lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
	}, <-subscribeEventChan)
}

func (ts *BackendTestSuite) TearDownTest() {
	assert := require.New(ts.T())
	assert.NoError(ts.backend.Stop())
}

func (ts *BackendTestSuite) TestGatewayStats() {
	assert := require.New(ts.T())
	gatewayStatsChan := make(chan gw.GatewayStats, 1)
	ts.backend.gatewayStatsFunc = func(pl gw.GatewayStats) {
		gatewayStatsChan <- pl
	}

	stats := gw.GatewayStats{
		GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	b, err := proto.Marshal(&stats)
	assert.NoError(err)

	assert.NoError(ts.pubSock.SendMulti(zmq4.Msg{
		Frames: [][]byte{
			[]byte("stats"),
			b,
		},
	}))

	recv := <-gatewayStatsChan
	assert.True(proto.Equal(&stats, &recv))
}

func (ts *BackendTestSuite) TestUplinkFrame() {
	assert := require.New(ts.T())
	uplinkFrameChan := make(chan gw.UplinkFrame, 1)
	ts.backend.uplinkFrameFunc = func(pl gw.UplinkFrame) {
		uplinkFrameChan <- pl
	}

	uf := gw.UplinkFrame{
		PhyPayload: []byte{1, 2, 3, 4},
		RxInfo: &gw.UplinkRXInfo{
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}
	b, err := proto.Marshal(&uf)
	assert.NoError(err)

	assert.NoError(ts.pubSock.SendMulti(zmq4.Msg{
		Frames: [][]byte{
			[]byte("up"),
			b,
		},
	}))

	recv := <-uplinkFrameChan
	assert.True(proto.Equal(&uf, &recv))
}

func (ts *BackendTestSuite) TestSendDownlinkFrame() {
	assert := require.New(ts.T())
	txAckChan := make(chan gw.DownlinkTXAck, 1)
	ts.backend.downlinkTxAckFunc = func(pl gw.DownlinkTXAck) {
		txAckChan <- pl
	}

	down := gw.DownlinkFrame{
		GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	downB, err := proto.Marshal(&down)
	assert.NoError(err)

	ack := gw.DownlinkTXAck{
		GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	ackB, err := proto.Marshal(&ack)
	assert.NoError(err)

	go func() {
		msg, err := ts.repSock.Recv()
		assert.NoError(err)
		assert.Equal("down", string(msg.Frames[0]))
		assert.Equal(downB, msg.Frames[1])
		assert.NoError(ts.repSock.Send(zmq4.NewMsg(ackB)))
	}()

	assert.NoError(ts.backend.SendDownlinkFrame(down))

	recv := <-txAckChan
	assert.True(proto.Equal(&ack, &recv))
}

func (ts *BackendTestSuite) TestApplyConfiguration() {
	assert := require.New(ts.T())

	config := gw.GatewayConfiguration{
		GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Version:   "config-a",
	}
	configB, err := proto.Marshal(&config)
	assert.NoError(err)

	go func() {
		msg, err := ts.repSock.Recv()
		assert.NoError(err)
		assert.Equal("config", string(msg.Frames[0]))
		assert.Equal(configB, msg.Frames[1])
		assert.NoError(ts.repSock.Send(zmq4.NewMsg([]byte{})))
	}()

	assert.NoError(ts.backend.ApplyConfiguration(config))
}

func TestBackend(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
