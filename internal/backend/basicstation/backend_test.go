package basicstation

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/brocaar/chirpstack-api/go/v3/common"
	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/basicstation/structs"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
)

type BackendTestSuite struct {
	suite.Suite

	backend  *Backend
	wsClient *websocket.Conn
	wsAddr   string
}

func (ts *BackendTestSuite) SetupSuite() {
	log.SetLevel(log.ErrorLevel)
}

func (ts *BackendTestSuite) SetupTest() {
	var err error
	assert := require.New(ts.T())

	var conf config.Config
	conf.Backend.Type = "basic_station"
	conf.Backend.BasicStation.Bind = "127.0.0.1:0"
	conf.Filters.NetIDs = []string{"010203"}
	conf.Filters.JoinEUIs = [][2]string{{"0000000000000000", "0102030405060708"}}
	conf.Backend.BasicStation.Region = "EU868"
	conf.Backend.BasicStation.FrequencyMin = 867000000
	conf.Backend.BasicStation.FrequencyMax = 869000000
	conf.Backend.BasicStation.StatsInterval = 30 * time.Second
	conf.Backend.BasicStation.PingInterval = time.Minute
	conf.Backend.BasicStation.ReadTimeout = 2 * time.Minute
	conf.Backend.BasicStation.WriteTimeout = time.Second

	ts.backend, err = NewBackend(conf)
	assert.NoError(err)

	ts.wsAddr = ts.backend.ln.Addr().String()

	d := &websocket.Dialer{}

	ts.wsClient, _, err = d.Dial(fmt.Sprintf("ws://%s/gateway/0102030405060708", ts.wsAddr), nil)
	assert.NoError(err)

	event := <-ts.backend.GetSubscribeEventChan()
	assert.Equal(events.Subscribe{Subscribe: true, GatewayID: lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}, event)
}

func (ts *BackendTestSuite) TearDownTest() {
	assert := require.New(ts.T())
	assert.NoError(ts.wsClient.Close())

	event := <-ts.backend.GetSubscribeEventChan()
	assert.Equal(events.Subscribe{Subscribe: false, GatewayID: lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}, event)

	assert.NoError(ts.backend.Close())
}

func (ts *BackendTestSuite) TestRouterInfo() {
	assert := require.New(ts.T())

	d := &websocket.Dialer{}

	ws, _, err := d.Dial(fmt.Sprintf("ws://%s/router-info", ts.wsAddr), nil)
	assert.NoError(err)
	defer ws.Close()

	ri := structs.RouterInfoRequest{
		Router: structs.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
	}

	assert.NoError(ws.WriteJSON(ri))

	var resp structs.RouterInfoResponse
	assert.NoError(ws.ReadJSON(&resp))

	assert.Equal(structs.RouterInfoResponse{
		Router: structs.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Muxs:   structs.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		URI:    fmt.Sprintf("ws://%s/gateway/0102030405060708", ts.wsAddr),
	}, resp)
}

func (ts *BackendTestSuite) TestVersion() {
	assert := require.New(ts.T())
	ts.backend.routerConfig = structs.RouterConfig{
		MessageType: structs.RouterConfigMessage,
	}

	ver := structs.Version{
		MessageType: structs.VersionMessage,
		Protocol:    2,
	}

	assert.NoError(ts.wsClient.WriteJSON(ver))

	var routerConfig structs.RouterConfig
	assert.NoError(ts.wsClient.ReadJSON(&routerConfig))

	assert.Equal(ts.backend.routerConfig, routerConfig)
}

func (ts *BackendTestSuite) TestUplinkDataFrame() {
	assert := require.New(ts.T())

	upf := structs.UplinkDataFrame{
		RadioMetaData: structs.RadioMetaData{
			DR:        5,
			Frequency: 868100000,
			UpInfo: structs.RadioMetaDataUpInfo{
				RCtx:  1,
				XTime: 2,
				RSSI:  120,
				SNR:   5.5,
			},
		},
		MessageType: structs.UplinkDataFrameMessage,
		MHDR:        0x40, // unconfirmed data-up
		DevAddr:     -10,
		FCtrl:       0x80, // ADR
		FCnt:        400,
		FOpts:       "0102", // invalid, but for the purpose of testing
		MIC:         -20,
		FPort:       -1,
	}

	assert.NoError(ts.wsClient.WriteJSON(upf))

	uplinkFrame := <-ts.backend.GetUplinkFrameChan()

	assert.Len(uplinkFrame.RxInfo.UplinkId, 16)
	uplinkFrame.RxInfo.UplinkId = nil

	assert.Equal(gw.UplinkFrame{
		PhyPayload: []byte{0x40, 0xf6, 0xff, 0xff, 0x0ff, 0x80, 0x90, 0x01, 0x01, 0x02, 0xec, 0xff, 0xff, 0xff},
		TxInfo: &gw.UplinkTXInfo{
			Frequency:  868100000,
			Modulation: common.Modulation_LORA,
			ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
				LoraModulationInfo: &gw.LoRaModulationInfo{
					Bandwidth:       125,
					SpreadingFactor: 7,
					CodeRate:        "4/5",
				},
			},
		},
		RxInfo: &gw.UplinkRXInfo{
			GatewayId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			Rssi:      120,
			LoraSnr:   5.5,
			Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}, uplinkFrame)

	rx, ok := ts.backend.statsCache.Get("0102030405060708:rx")
	assert.True(ok)
	assert.Equal(uint32(1), rx)

	rxOK, ok := ts.backend.statsCache.Get("0102030405060708:rxOK")
	assert.True(ok)
	assert.Equal(uint32(1), rxOK)
}

func (ts *BackendTestSuite) TestJoinRequest() {
	assert := require.New(ts.T())

	jr := structs.JoinRequest{
		RadioMetaData: structs.RadioMetaData{
			DR:        5,
			Frequency: 868100000,
			UpInfo: structs.RadioMetaDataUpInfo{
				RCtx:  1,
				XTime: 2,
				RSSI:  120,
				SNR:   5.5,
			},
		},

		MessageType: structs.JoinRequestMessage,
		MHDR:        0x00,
		JoinEUI:     structs.EUI64{0x02, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		DevEUI:      structs.EUI64{0x03, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		DevNonce:    20,
		MIC:         -10,
	}

	assert.NoError(ts.wsClient.WriteJSON(jr))

	uplinkFrame := <-ts.backend.GetUplinkFrameChan()

	assert.Len(uplinkFrame.RxInfo.UplinkId, 16)
	uplinkFrame.RxInfo.UplinkId = nil

	assert.Equal(gw.UplinkFrame{
		PhyPayload: []byte{0x00, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x02, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x03, 0x14, 0x00, 0xf6, 0xff, 0xff, 0xff},
		TxInfo: &gw.UplinkTXInfo{
			Frequency:  868100000,
			Modulation: common.Modulation_LORA,
			ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
				LoraModulationInfo: &gw.LoRaModulationInfo{
					Bandwidth:       125,
					SpreadingFactor: 7,
					CodeRate:        "4/5",
				},
			},
		},
		RxInfo: &gw.UplinkRXInfo{
			GatewayId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			Rssi:      120,
			LoraSnr:   5.5,
			Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}, uplinkFrame)

	rx, ok := ts.backend.statsCache.Get("0102030405060708:rx")
	assert.True(ok)
	assert.Equal(uint32(1), rx)

	rxOK, ok := ts.backend.statsCache.Get("0102030405060708:rxOK")
	assert.True(ok)
	assert.Equal(uint32(1), rxOK)
}

func (ts *BackendTestSuite) TestProprietaryDataFrame() {
	assert := require.New(ts.T())

	propf := structs.UplinkProprietaryFrame{
		RadioMetaData: structs.RadioMetaData{
			DR:        5,
			Frequency: 868100000,
			UpInfo: structs.RadioMetaDataUpInfo{
				RCtx:  1,
				XTime: 2,
				RSSI:  120,
				SNR:   5.5,
			},
		},
		MessageType: structs.ProprietaryDataFrameMessage,
		FRMPayload:  "01020304",
	}

	assert.NoError(ts.wsClient.WriteJSON(propf))

	uplinkFrame := <-ts.backend.GetUplinkFrameChan()

	assert.Len(uplinkFrame.RxInfo.UplinkId, 16)
	uplinkFrame.RxInfo.UplinkId = nil

	assert.Equal(gw.UplinkFrame{
		PhyPayload: []byte{0x01, 0x02, 0x03, 0x04},
		TxInfo: &gw.UplinkTXInfo{
			Frequency:  868100000,
			Modulation: common.Modulation_LORA,
			ModulationInfo: &gw.UplinkTXInfo_LoraModulationInfo{
				LoraModulationInfo: &gw.LoRaModulationInfo{
					Bandwidth:       125,
					SpreadingFactor: 7,
					CodeRate:        "4/5",
				},
			},
		},
		RxInfo: &gw.UplinkRXInfo{
			GatewayId: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			Rssi:      120,
			LoraSnr:   5.5,
			Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}, uplinkFrame)

	rx, ok := ts.backend.statsCache.Get("0102030405060708:rx")
	assert.True(ok)
	assert.Equal(uint32(1), rx)

	rxOK, ok := ts.backend.statsCache.Get("0102030405060708:rxOK")
	assert.True(ok)
	assert.Equal(uint32(1), rxOK)
}

func (ts *BackendTestSuite) TestDownlinkTransmitted() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	ts.backend.diidCache.SetDefault("12345", id[:])

	dtx := structs.DownlinkTransmitted{
		MessageType: structs.DownlinkTransmittedMessage,
		DIID:        12345,
	}

	assert.NoError(ts.wsClient.WriteJSON(dtx))

	txAck := <-ts.backend.GetDownlinkTXAckChan()

	assert.Equal(gw.DownlinkTXAck{
		GatewayId:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		Token:      12345,
		DownlinkId: id[:],
	}, txAck)

	// this variable is not yet stored
	_, ok := ts.backend.statsCache.Get("0102030405060708:tx")
	assert.False(ok)

	txOK, ok := ts.backend.statsCache.Get("0102030405060708:txOK")
	assert.True(ok)
	assert.Equal(uint32(1), txOK)
}

func (ts *BackendTestSuite) TestSendDownlinkFrame() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	err = ts.backend.SendDownlinkFrame(gw.DownlinkFrame{
		Token:      1234,
		DownlinkId: id[:],
		GatewayId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Items: []*gw.DownlinkFrameItem{
			{
				PhyPayload: []byte{1, 2, 3, 4},
				TxInfo: &gw.DownlinkTXInfo{
					Frequency:  868100000,
					Power:      14,
					Modulation: common.Modulation_LORA,
					ModulationInfo: &gw.DownlinkTXInfo_LoraModulationInfo{
						LoraModulationInfo: &gw.LoRaModulationInfo{
							Bandwidth:             125,
							SpreadingFactor:       10,
							CodeRate:              "4/5",
							PolarizationInversion: true,
						},
					},
					Timing: gw.DownlinkTiming_DELAY,
					TimingInfo: &gw.DownlinkTXInfo_DelayTimingInfo{
						DelayTimingInfo: &gw.DelayTimingInfo{
							Delay: ptypes.DurationProto(time.Second),
						},
					},
					Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
				},
			},
		},
	})
	assert.NoError(err)

	idResp, ok := ts.backend.diidCache.Get("1234")
	assert.True(ok)
	assert.Equal(id[:], idResp)

	var df structs.DownlinkFrame
	assert.NoError(ts.wsClient.ReadJSON(&df))

	delay1 := 1
	dr2 := 2
	freq := uint32(868100000)
	rCtx := uint64(3)
	xTime := uint64(4)

	assert.Equal(structs.DownlinkFrame{
		MessageType: structs.DownlinkMessage,
		DevEui:      "01-01-01-01-01-01-01-01",
		DC:          0,
		DIID:        1234,
		Priority:    1,
		PDU:         "01020304",
		RCtx:        &rCtx,
		XTime:       &xTime,
		RxDelay:     &delay1,
		RX1DR:       &dr2,
		RX1Freq:     &freq,
	}, df)

	tx, ok := ts.backend.statsCache.Get("0102030405060708:tx")
	assert.True(ok)
	assert.Equal(uint32(1), tx)

	// this variable is not yet stored
	_, ok = ts.backend.statsCache.Get("0102030405060708:txOK")
	assert.False(ok)
}

func (ts *BackendTestSuite) TestRawPacketForwarderCommand() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	ts.T().Run("JSON", func(t *testing.T) {
		assert := require.New(t)
		pl := gw.RawPacketForwarderCommand{
			GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
			RawId:     id[:],
			Payload:   []byte(`{"foo": "bar"}`),
		}
		assert.NoError(ts.backend.RawPacketForwarderCommand(pl))

		mt, msg, err := ts.wsClient.ReadMessage()
		assert.NoError(err)
		assert.Equal(websocket.TextMessage, mt)
		assert.Equal(pl.Payload, msg)
	})

	ts.T().Run("Binary", func(t *testing.T) {
		assert := require.New(t)
		pl := gw.RawPacketForwarderCommand{
			GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
			RawId:     id[:],
			Payload:   []byte{0x01, 0x02, 0x03, 0x04},
		}
		assert.NoError(ts.backend.RawPacketForwarderCommand(pl))

		mt, msg, err := ts.wsClient.ReadMessage()
		assert.NoError(err)
		assert.Equal(websocket.BinaryMessage, mt)
		assert.Equal(pl.Payload, msg)
	})
}

func (ts *BackendTestSuite) TestRawPacketForwarderEvent() {
	ts.T().Run("Binary", func(t *testing.T) {
		assert := require.New(t)

		assert.NoError(ts.wsClient.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02, 0x03, 0x04}))

		pl := <-ts.backend.GetRawPacketForwarderEventChan()
		assert.Equal([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, pl.GatewayId)
		assert.NotNil(pl.RawId)
		assert.Equal([]byte{0x01, 0x02, 0x03, 0x04}, pl.Payload)
	})

	ts.T().Run("JSON rmtsh", func(t *testing.T) {
		assert := require.New(t)

		jsonMsg := `{
		  "msgtype"  : "rmtsh",
		  "rmtsh"    : [
			{
			  "user"     : "foo",
			  "started"  : true,
			  "age"      : 1,
			  "pid"      : 2
			}
		  ]
		}`

		assert.NoError(ts.wsClient.WriteMessage(websocket.TextMessage, []byte(jsonMsg)))

		pl := <-ts.backend.GetRawPacketForwarderEventChan()
		assert.Equal([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, pl.GatewayId)
		assert.NotNil(pl.RawId)
		assert.Equal([]byte(jsonMsg), pl.Payload)
	})
}

func TestBackend(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
