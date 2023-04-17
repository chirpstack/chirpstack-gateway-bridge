package basicstation

import (
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/basicstation/structs"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/gps"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
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
	conf.Backend.BasicStation.StatsInterval = 30 * time.Second
	conf.Backend.BasicStation.Region = "EU868"
	conf.Backend.BasicStation.FrequencyMin = 867000000
	conf.Backend.BasicStation.FrequencyMax = 869000000
	conf.Backend.BasicStation.PingInterval = time.Minute
	conf.Backend.BasicStation.ReadTimeout = 2 * time.Minute
	conf.Backend.BasicStation.WriteTimeout = time.Second

	ts.backend, err = NewBackend(conf)
	assert.NoError(err)

	subscribeChan := make(chan events.Subscribe, 1)
	ts.backend.gateways.subscribeEventFunc = func(pl events.Subscribe) {
		subscribeChan <- pl
	}
	assert.NoError(ts.backend.Start())

	ts.wsAddr = ts.backend.ln.Addr().String()

	d := &websocket.Dialer{}

	ts.wsClient, _, err = d.Dial(fmt.Sprintf("ws://%s/gateway/0102030405060708", ts.wsAddr), nil)
	assert.NoError(err)

	event := <-subscribeChan
	assert.Equal(events.Subscribe{Subscribe: true, GatewayID: lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}, event)
}

func (ts *BackendTestSuite) TearDownTest() {
	subscribeChan := make(chan events.Subscribe, 1)
	ts.backend.gateways.subscribeEventFunc = func(pl events.Subscribe) {
		subscribeChan <- pl
	}

	assert := require.New(ts.T())
	assert.NoError(ts.wsClient.Close())

	event := <-subscribeChan
	assert.Equal(events.Subscribe{Subscribe: false, GatewayID: lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}, event)

	assert.NoError(ts.backend.Stop())
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

	ver := structs.Version{
		MessageType: structs.VersionMessage,
		Protocol:    2,
	}

	assert.NoError(ts.wsClient.WriteJSON(ver))

	var routerConfig structs.RouterConfig
	assert.NoError(ts.wsClient.ReadJSON(&routerConfig))

	routerConfig, err := ts.backend.getRouterConfig()
	assert.NoError(err)

	assert.Equal(routerConfig, routerConfig)
}

func (ts *BackendTestSuite) TestUplinkDataFrame() {
	assert := require.New(ts.T())

	uplinkFrameChan := make(chan *gw.UplinkFrame, 1)
	ts.backend.uplinkFrameFunc = func(pl *gw.UplinkFrame) {
		uplinkFrameChan <- pl
	}

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

	uplinkFrame := <-uplinkFrameChan

	assert.NotEqual(uplinkFrame.RxInfo.UplinkId, 0)
	uplinkFrame.RxInfo.UplinkId = 0

	assert.True(proto.Equal(&gw.UplinkFrame{
		PhyPayload: []byte{0x40, 0xf6, 0xff, 0xff, 0x0ff, 0x80, 0x90, 0x01, 0x01, 0x02, 0xec, 0xff, 0xff, 0xff},
		TxInfo: &gw.UplinkTxInfo{
			Frequency: 868100000,
			Modulation: &gw.Modulation{
				Parameters: &gw.Modulation_Lora{
					Lora: &gw.LoraModulationInfo{
						Bandwidth:       125000,
						SpreadingFactor: 7,
						CodeRate:        gw.CodeRate_CR_4_5,
					},
				},
			},
		},
		RxInfo: &gw.UplinkRxInfo{
			GatewayId: "0102030405060708",
			Rssi:      120,
			Snr:       5.5,
			Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}, uplinkFrame))

	conn, err := ts.backend.gateways.get(lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	assert.NoError(err)

	stats := conn.stats.ExportStats()
	assert.True(proto.Equal(&gw.GatewayStats{
		RxPacketsReceived:   1,
		RxPacketsReceivedOk: 1,
		RxPacketsPerFrequency: map[uint32]uint32{
			868100000: 1,
		},
		RxPacketsPerModulation: []*gw.PerModulationCount{
			{
				Count: 1,
				Modulation: &gw.Modulation{
					Parameters: &gw.Modulation_Lora{
						Lora: &gw.LoraModulationInfo{
							Bandwidth:       125000,
							SpreadingFactor: 7,
							CodeRate:        gw.CodeRate_CR_4_5,
						},
					},
				},
			},
		},
	}, stats))
}

func (ts *BackendTestSuite) TestJoinRequest() {
	assert := require.New(ts.T())

	uplinkFrameChan := make(chan *gw.UplinkFrame, 1)
	ts.backend.uplinkFrameFunc = func(pl *gw.UplinkFrame) {
		uplinkFrameChan <- pl
	}

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

	uplinkFrame := <-uplinkFrameChan

	assert.NotEqual(uplinkFrame.RxInfo.UplinkId, 0)
	uplinkFrame.RxInfo.UplinkId = 0

	assert.True(proto.Equal(&gw.UplinkFrame{
		PhyPayload: []byte{0x00, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x02, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x03, 0x14, 0x00, 0xf6, 0xff, 0xff, 0xff},
		TxInfo: &gw.UplinkTxInfo{
			Frequency: 868100000,
			Modulation: &gw.Modulation{
				Parameters: &gw.Modulation_Lora{
					Lora: &gw.LoraModulationInfo{
						Bandwidth:       125000,
						SpreadingFactor: 7,
						CodeRate:        gw.CodeRate_CR_4_5,
					},
				},
			},
		},
		RxInfo: &gw.UplinkRxInfo{
			GatewayId: "0102030405060708",
			Rssi:      120,
			Snr:       5.5,
			Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}, uplinkFrame))

	conn, err := ts.backend.gateways.get(lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	assert.NoError(err)

	stats := conn.stats.ExportStats()
	assert.True(proto.Equal(&gw.GatewayStats{
		RxPacketsReceived:   1,
		RxPacketsReceivedOk: 1,
		RxPacketsPerFrequency: map[uint32]uint32{
			868100000: 1,
		},
		RxPacketsPerModulation: []*gw.PerModulationCount{
			{
				Count: 1,
				Modulation: &gw.Modulation{
					Parameters: &gw.Modulation_Lora{
						Lora: &gw.LoraModulationInfo{
							Bandwidth:       125000,
							SpreadingFactor: 7,
							CodeRate:        gw.CodeRate_CR_4_5,
						},
					},
				},
			},
		},
	}, stats))
}

func (ts *BackendTestSuite) TestProprietaryDataFrame() {
	assert := require.New(ts.T())

	uplinkFrameChan := make(chan *gw.UplinkFrame)
	ts.backend.uplinkFrameFunc = func(pl *gw.UplinkFrame) {
		uplinkFrameChan <- pl
	}

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

	uplinkFrame := <-uplinkFrameChan

	assert.NotEqual(uplinkFrame.RxInfo.UplinkId, 0)
	uplinkFrame.RxInfo.UplinkId = 0

	assert.True(proto.Equal(&gw.UplinkFrame{
		PhyPayload: []byte{0x01, 0x02, 0x03, 0x04},
		TxInfo: &gw.UplinkTxInfo{
			Frequency: 868100000,
			Modulation: &gw.Modulation{
				Parameters: &gw.Modulation_Lora{
					Lora: &gw.LoraModulationInfo{
						Bandwidth:       125000,
						SpreadingFactor: 7,
						CodeRate:        gw.CodeRate_CR_4_5,
					},
				},
			},
		},
		RxInfo: &gw.UplinkRxInfo{
			GatewayId: "0102030405060708",
			Rssi:      120,
			Snr:       5.5,
			Context:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			CrcStatus: gw.CRCStatus_CRC_OK,
		},
	}, uplinkFrame))

	conn, err := ts.backend.gateways.get(lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	assert.NoError(err)

	stats := conn.stats.ExportStats()
	assert.True(proto.Equal(&gw.GatewayStats{
		RxPacketsReceived:   1,
		RxPacketsReceivedOk: 1,
		RxPacketsPerFrequency: map[uint32]uint32{
			868100000: 1,
		},
		RxPacketsPerModulation: []*gw.PerModulationCount{
			{
				Count: 1,
				Modulation: &gw.Modulation{
					Parameters: &gw.Modulation_Lora{
						Lora: &gw.LoraModulationInfo{
							Bandwidth:       125000,
							SpreadingFactor: 7,
							CodeRate:        gw.CodeRate_CR_4_5,
						},
					},
				},
			},
		},
	}, stats))
}

func (ts *BackendTestSuite) TestRequestTimesync() {
	assert := require.New(ts.T())
	ts.backend.timesyncInterval = time.Hour
	gatewayID := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}

	lastTimesyncBefore, err := ts.backend.gateways.getLastTimesync(gatewayID)
	assert.NoError(err)

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

	var timesyncReq structs.TimeSyncGPSTimeTransfer
	assert.NoError(ts.wsClient.ReadJSON(&timesyncReq))

	assert.EqualValues(timesyncReq.XTime, 2)
	assert.True(timesyncReq.GPSTime > 0)

	lastTimesyncAfter, err := ts.backend.gateways.getLastTimesync(gatewayID)
	assert.NoError(err)

	assert.NotEqual(lastTimesyncBefore, lastTimesyncAfter)
}

func (ts *BackendTestSuite) TestDownlinkTransmitted() {
	assert := require.New(ts.T())

	txAckChan := make(chan *gw.DownlinkTxAck, 1)
	ts.backend.downlinkTxAckFunc = func(pl *gw.DownlinkTxAck) {
		txAckChan <- pl
	}

	df := gw.DownlinkFrame{
		DownlinkId: 12345,
		GatewayId:  "0102030405060708",
		Items: []*gw.DownlinkFrameItem{
			{
				PhyPayload: []byte{1, 2, 3, 4},
				TxInfo: &gw.DownlinkTxInfo{
					Frequency: 868100000,
					Power:     14,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:             125000,
								SpreadingFactor:       10,
								CodeRate:              gw.CodeRate_CR_4_5,
								PolarizationInversion: true,
							},
						},
					},
					Timing: &gw.Timing{
						Parameters: &gw.Timing_Delay{
							Delay: &gw.DelayTimingInfo{
								Delay: durationpb.New(time.Second),
							},
						},
					},
					Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
				},
			},
		},
	}

	ts.backend.diidCache.SetDefault("12345", &df)

	dtx := structs.DownlinkTransmitted{
		MessageType: structs.DownlinkTransmittedMessage,
		DIID:        12345,
	}

	assert.NoError(ts.wsClient.WriteJSON(dtx))

	txAck := <-txAckChan

	assert.Equal(&gw.DownlinkTxAck{
		GatewayId:  "0102030405060708",
		DownlinkId: 12345,
		Items: []*gw.DownlinkTxAckItem{
			{
				Status: gw.TxAckStatus_OK,
			},
		},
	}, txAck)

	conn, err := ts.backend.gateways.get(lorawan.EUI64{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08})
	assert.NoError(err)

	stats := conn.stats.ExportStats()
	assert.True(proto.Equal(&gw.GatewayStats{
		TxPacketsReceived: 1,
		TxPacketsEmitted:  1,
		TxPacketsPerFrequency: map[uint32]uint32{
			868100000: 1,
		},
		TxPacketsPerModulation: []*gw.PerModulationCount{
			{
				Count: 1,
				Modulation: &gw.Modulation{
					Parameters: &gw.Modulation_Lora{
						Lora: &gw.LoraModulationInfo{
							Bandwidth:             125000,
							SpreadingFactor:       10,
							CodeRate:              gw.CodeRate_CR_4_5,
							PolarizationInversion: true,
						},
					},
				},
			},
		},
		TxPacketsPerStatus: map[string]uint32{
			"OK": 1,
		},
	}, stats))
}

func (ts *BackendTestSuite) TestSendDownlinkFrame() {
	assert := require.New(ts.T())

	pl := gw.DownlinkFrame{
		DownlinkId: 1234,
		GatewayId:  "0102030405060708",
		Items: []*gw.DownlinkFrameItem{
			{
				PhyPayload: []byte{1, 2, 3, 4},
				TxInfo: &gw.DownlinkTxInfo{
					Frequency: 868100000,
					Power:     14,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:             125000,
								SpreadingFactor:       10,
								CodeRate:              gw.CodeRate_CR_4_5,
								PolarizationInversion: true,
							},
						},
					},
					Timing: &gw.Timing{
						Parameters: &gw.Timing_Delay{
							Delay: &gw.DelayTimingInfo{
								Delay: durationpb.New(time.Second),
							},
						},
					},
					Context: []byte{0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4},
				},
			},
		},
	}

	err := ts.backend.SendDownlinkFrame(&pl)
	assert.NoError(err)

	idResp, ok := ts.backend.diidCache.Get("1234")
	assert.True(ok)
	assert.Equal(&pl, idResp)

	var df structs.DownlinkFrame
	assert.NoError(ts.wsClient.ReadJSON(&df))

	delay1 := 1
	dr2 := 2
	freq := uint32(868100000)
	rCtx := uint64(3)
	xTime := uint64(4)

	assert.NotNil(df.MuxTime)
	df.MuxTime = nil

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

	/*
		tx, ok := ts.backend.statsCache.Get("0102030405060708:tx")
		assert.True(ok)
		assert.Equal(uint32(1), tx)

		// this variable is not yet stored
		_, ok = ts.backend.statsCache.Get("0102030405060708:txOK")
		assert.False(ok)
	*/
}

func (ts *BackendTestSuite) TestRawPacketForwarderCommand() {
	ts.T().Run("JSON", func(t *testing.T) {
		assert := require.New(t)
		pl := gw.RawPacketForwarderCommand{
			GatewayId: "0102030405060708",
			Payload:   []byte(`{"foo": "bar"}`),
		}
		assert.NoError(ts.backend.RawPacketForwarderCommand(&pl))

		mt, msg, err := ts.wsClient.ReadMessage()
		assert.NoError(err)
		assert.Equal(websocket.TextMessage, mt)
		assert.Equal(pl.Payload, msg)
	})

	ts.T().Run("Binary", func(t *testing.T) {
		assert := require.New(t)
		pl := gw.RawPacketForwarderCommand{
			GatewayId: "0102030405060708",
			Payload:   []byte{0x01, 0x02, 0x03, 0x04},
		}
		assert.NoError(ts.backend.RawPacketForwarderCommand(&pl))

		mt, msg, err := ts.wsClient.ReadMessage()
		assert.NoError(err)
		assert.Equal(websocket.BinaryMessage, mt)
		assert.Equal(pl.Payload, msg)
	})
}

func (ts *BackendTestSuite) TestRawPacketForwarderEvent() {
	rawPacketForwarderEventChan := make(chan *gw.RawPacketForwarderEvent, 1)
	ts.backend.rawPacketForwarderEventFunc = func(pl *gw.RawPacketForwarderEvent) {
		rawPacketForwarderEventChan <- pl
	}

	ts.T().Run("Binary", func(t *testing.T) {
		assert := require.New(t)

		assert.NoError(ts.wsClient.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02, 0x03, 0x04}))

		pl := <-rawPacketForwarderEventChan
		assert.Equal("0102030405060708", pl.GatewayId)
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

		pl := <-rawPacketForwarderEventChan
		assert.Equal("0102030405060708", pl.GatewayId)
		assert.Equal([]byte(jsonMsg), pl.Payload)
	})
}

func (ts *BackendTestSuite) TestTimeSync() {
	assert := require.New(ts.T())

	startGPS := int64(gps.Time(time.Now()).TimeSinceGPSEpoch() / time.Microsecond)
	tsync := structs.TimeSyncRequest{
		MessageType: structs.TimeSyncMessage,
		TxTime:      112233445566,
	}

	assert.NoError(ts.wsClient.WriteJSON(&tsync))

	var tsresp structs.TimeSyncResponse
	assert.NoError(ts.wsClient.ReadJSON(&tsresp))

	assert.Equal(tsync.MessageType, tsresp.MessageType)
	assert.Equal(tsync.TxTime, tsresp.TxTime)

	endGPS := int64(gps.Time(time.Now()).TimeSinceGPSEpoch() / time.Microsecond)
	assert.True(tsresp.GPSTime >= startGPS && tsresp.GPSTime <= endGPS)
}

func TestBackend(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
