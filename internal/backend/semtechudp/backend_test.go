package semtechudp

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/semtechudp/packets"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/chirpstack/chirpstack/api/go/v4/common"
	"github.com/chirpstack/chirpstack/api/go/v4/gw"
)

type BackendTestSuite struct {
	suite.Suite

	tempDir        string
	backend        *Backend
	backendUDPAddr *net.UDPAddr
	gwUDPConn      *net.UDPConn
}

func (ts *BackendTestSuite) SetupSuite() {
	log.SetLevel(log.ErrorLevel)
}

func (ts *BackendTestSuite) SetupTest() {
	var err error
	assert := require.New(ts.T())

	ts.tempDir, err = ioutil.TempDir("", "test")
	assert.NoError(err)

	var conf config.Config
	conf.Backend.SemtechUDP.UDPBind = "127.0.0.1:0"

	ts.backend, err = NewBackend(conf)
	assert.NoError(err)
	assert.NoError(ts.backend.Start())

	ts.backendUDPAddr, err = net.ResolveUDPAddr("udp", ts.backend.conn.LocalAddr().String())
	assert.NoError(err)

	gwAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	assert.NoError(err)

	ts.gwUDPConn, err = net.ListenUDP("udp", gwAddr)
	assert.NoError(err)
	assert.NoError(ts.gwUDPConn.SetDeadline(time.Now().Add(time.Second)))
}

func (ts *BackendTestSuite) TearDownTest() {
	os.RemoveAll(ts.tempDir)
	ts.backend.Stop()
	ts.gwUDPConn.Close()
}

func (ts *BackendTestSuite) TestPullData() {
	ts.T().Run("Send PullData", func(t *testing.T) {
		assert := require.New(t)

		p := packets.PullDataPacket{
			ProtocolVersion: packets.ProtocolVersion2,
			RandomToken:     12345,
			GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		}
		b, err := p.MarshalBinary()
		assert.NoError(err)

		_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
		assert.NoError(err)

		t.Run("Receive PullACK", func(t *testing.T) {
			assert := require.New(t)

			buf := make([]byte, 65507)
			i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
			assert.NoError(err)
			var ack packets.PullACKPacket
			assert.NoError(ack.UnmarshalBinary(buf[:i]))
			assert.Equal(p.RandomToken, ack.RandomToken)
			assert.Equal(p.ProtocolVersion, ack.ProtocolVersion)
		})
	})
}

func (ts *BackendTestSuite) TestTXAck() {
	testTable := []struct {
		Name          string
		GatewayPacket packets.TXACKPacket
		BackendPacket *gw.DownlinkTxAck
	}{
		{
			Name: "no error",
			GatewayPacket: packets.TXACKPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     12345,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			},
			BackendPacket: &gw.DownlinkTxAck{
				GatewayId:  "0102030405060708",
				DownlinkId: 12345,
				Items: []*gw.DownlinkTxAckItem{
					{
						Status: gw.TxAckStatus_OK,
					},
				},
			},
		},
		{
			Name: "error NONE",
			GatewayPacket: packets.TXACKPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     12345,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: &packets.TXACKPayload{
					TXPKACK: packets.TXPKACK{
						Error: "NONE",
					},
				},
			},
			BackendPacket: &gw.DownlinkTxAck{
				GatewayId:  "0102030405060708",
				DownlinkId: 12345,
				Items: []*gw.DownlinkTxAckItem{
					{
						Status: gw.TxAckStatus_OK,
					},
				},
			},
		},
		{
			Name: "error",
			GatewayPacket: packets.TXACKPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     12345,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: &packets.TXACKPayload{
					TXPKACK: packets.TXPKACK{
						Error: "TX_FREQ",
					},
				},
			},
			BackendPacket: &gw.DownlinkTxAck{
				GatewayId:  "0102030405060708",
				DownlinkId: 12345,
				Items: []*gw.DownlinkTxAckItem{
					{
						Status: gw.TxAckStatus_TX_FREQ,
					},
				},
			},
		},
	}

	for _, test := range testTable {
		ts.T().Run(test.Name, func(t *testing.T) {
			assert := require.New(t)

			ackChan := make(chan *gw.DownlinkTxAck, 1)
			ts.backend.SetDownlinkTxAckFunc(func(pl *gw.DownlinkTxAck) {
				ackChan <- pl
			})

			ts.backend.cache.Set("12345:ack", make([]*gw.DownlinkTxAckItem, 1), cache.DefaultExpiration)
			ts.backend.cache.Set("12345:frame", &gw.DownlinkFrame{
				DownlinkId: 12345,
				Items: []*gw.DownlinkFrameItem{
					{},
				},
			}, cache.DefaultExpiration)
			ts.backend.cache.Set("12345:index", 0, cache.DefaultExpiration)

			b, err := test.GatewayPacket.MarshalBinary()
			assert.NoError(err)
			_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
			assert.NoError(err)

			ack := <-ackChan
			assert.True(proto.Equal(test.BackendPacket, ack))
		})
	}
}

func (ts *BackendTestSuite) TestTXAckRetryFailOK() {
	assert := require.New(ts.T())
	buf := make([]byte, 65507)

	// register gateway
	p := packets.PullDataPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
	}
	b, err := p.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
	assert.NoError(err)
	i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
	assert.NoError(err)
	var ack packets.PullACKPacket
	assert.NoError(ack.UnmarshalBinary(buf[:i]))
	assert.Equal(p.RandomToken, ack.RandomToken)
	assert.Equal(p.ProtocolVersion, ack.ProtocolVersion)

	// set cache
	ts.backend.cache.Set("12345:frame", &gw.DownlinkFrame{
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
								SpreadingFactor:       7,
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
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
			},
			{
				PhyPayload: []byte{1, 2, 3, 4, 5},
				TxInfo: &gw.DownlinkTxInfo{
					Frequency: 868100000,
					Power:     14,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:             125000,
								SpreadingFactor:       7,
								CodeRate:              gw.CodeRate_CR_4_5,
								PolarizationInversion: true,
							},
						},
					},
					Timing: &gw.Timing{
						Parameters: &gw.Timing_Delay{
							Delay: &gw.DelayTimingInfo{
								Delay: durationpb.New(time.Second * 2),
							},
						},
					},
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
			},
		},
		DownlinkId: 12345,
		GatewayId:  "0102030405060708",
	}, cache.DefaultExpiration)
	ts.backend.cache.Set("12345:index", 0, cache.DefaultExpiration)
	ts.backend.cache.Set("12345:ack", []*gw.DownlinkTxAckItem{
		{Status: gw.TxAckStatus_IGNORED},
		{Status: gw.TxAckStatus_IGNORED},
	}, cache.DefaultExpiration)

	// send a nack on the first downlink attempt
	ack1 := packets.TXACKPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Payload: &packets.TXACKPayload{
			TXPKACK: packets.TXPKACK{
				Error: "TX_FREQ",
			},
		},
	}
	ack1B, err := ack1.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(ack1B, ts.backendUDPAddr)
	assert.NoError(err)

	// validate udp packet
	i, _, err = ts.gwUDPConn.ReadFromUDP(buf)
	assert.NoError(err)
	tmst := uint32(3000000)
	var pullResp packets.PullRespPacket
	assert.NoError(pullResp.UnmarshalBinary(buf[:i]))
	assert.Equal(packets.PullRespPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		Payload: packets.PullRespPayload{
			TXPK: packets.TXPK{
				Tmst: &tmst,
				Freq: 868.1,
				RFCh: 0,
				Powe: 14,
				Modu: "LORA",
				DatR: packets.DatR{
					LoRa: "SF7BW125",
				},
				CodR: "4/5",
				IPol: true,
				Size: 5,
				Data: []byte{1, 2, 3, 4, 5},
			},
		},
	}, pullResp)

	// send an ack on the second downlink attempt
	ack2 := packets.TXACKPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Payload: &packets.TXACKPayload{
			TXPKACK: packets.TXPKACK{
				Error: "",
			},
		},
	}
	ack2B, err := ack2.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(ack2B, ts.backendUDPAddr)
	assert.NoError(err)

	ackChan := make(chan *gw.DownlinkTxAck, 1)
	ts.backend.SetDownlinkTxAckFunc(func(pl *gw.DownlinkTxAck) {
		ackChan <- pl
	})

	// validate final ack
	txAck := <-ackChan
	assert.Equal(&gw.DownlinkTxAck{
		GatewayId:  "0102030405060708",
		DownlinkId: 12345,
		Items: []*gw.DownlinkTxAckItem{
			{
				Status: gw.TxAckStatus_TX_FREQ,
			},
			{
				Status: gw.TxAckStatus_OK,
			},
		},
	}, txAck)
}

func (ts *BackendTestSuite) TestTXAckRetryFailFail() {
	assert := require.New(ts.T())
	buf := make([]byte, 65507)

	// register gateway
	p := packets.PullDataPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
	}
	b, err := p.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
	assert.NoError(err)
	i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
	assert.NoError(err)
	var ack packets.PullACKPacket
	assert.NoError(ack.UnmarshalBinary(buf[:i]))
	assert.Equal(p.RandomToken, ack.RandomToken)
	assert.Equal(p.ProtocolVersion, ack.ProtocolVersion)

	// set cache
	ts.backend.cache.Set("12345:frame", &gw.DownlinkFrame{
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
								SpreadingFactor:       7,
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
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
			},
			{
				PhyPayload: []byte{1, 2, 3, 4, 5},
				TxInfo: &gw.DownlinkTxInfo{
					Frequency: 868100000,
					Power:     14,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:             125000,
								SpreadingFactor:       7,
								CodeRate:              gw.CodeRate_CR_4_5,
								PolarizationInversion: true,
							},
						},
					},
					Timing: &gw.Timing{
						Parameters: &gw.Timing_Delay{
							Delay: &gw.DelayTimingInfo{
								Delay: durationpb.New(time.Second * 2),
							},
						},
					},
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
			},
		},
		DownlinkId: 12345,
		GatewayId:  "0102030405060708",
	}, cache.DefaultExpiration)
	ts.backend.cache.Set("12345:index", 0, cache.DefaultExpiration)
	ts.backend.cache.Set("12345:ack", []*gw.DownlinkTxAckItem{
		{Status: gw.TxAckStatus_IGNORED},
		{Status: gw.TxAckStatus_IGNORED},
	}, cache.DefaultExpiration)

	// Create ack channel
	ackChan := make(chan *gw.DownlinkTxAck, 1)
	ts.backend.SetDownlinkTxAckFunc(func(pl *gw.DownlinkTxAck) {
		ackChan <- pl
	})

	// send a nack on the first downlink attempt
	ack1 := packets.TXACKPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Payload: &packets.TXACKPayload{
			TXPKACK: packets.TXPKACK{
				Error: "TX_FREQ",
			},
		},
	}
	ack1B, err := ack1.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(ack1B, ts.backendUDPAddr)
	assert.NoError(err)

	// validate udp packet
	i, _, err = ts.gwUDPConn.ReadFromUDP(buf)
	assert.NoError(err)
	tmst := uint32(3000000)
	var pullResp packets.PullRespPacket
	assert.NoError(pullResp.UnmarshalBinary(buf[:i]))
	assert.Equal(packets.PullRespPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		Payload: packets.PullRespPayload{
			TXPK: packets.TXPK{
				Tmst: &tmst,
				Freq: 868.1,
				RFCh: 0,
				Powe: 14,
				Modu: "LORA",
				DatR: packets.DatR{
					LoRa: "SF7BW125",
				},
				CodR: "4/5",
				IPol: true,
				Size: 5,
				Data: []byte{1, 2, 3, 4, 5},
			},
		},
	}, pullResp)

	// send a nack on the second downlink attempt
	ack2 := packets.TXACKPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Payload: &packets.TXACKPayload{
			TXPKACK: packets.TXPKACK{
				Error: "TOO_LATE",
			},
		},
	}
	ack2B, err := ack2.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(ack2B, ts.backendUDPAddr)
	assert.NoError(err)

	// validate final ack
	txAck := <-ackChan
	assert.Equal(&gw.DownlinkTxAck{
		GatewayId:  "0102030405060708",
		DownlinkId: 12345,
		Items: []*gw.DownlinkTxAckItem{
			{
				Status: gw.TxAckStatus_TX_FREQ,
			},
			{
				Status: gw.TxAckStatus_TOO_LATE,
			},
		},
	}, txAck)
}

func (ts *BackendTestSuite) TestTXAckRetryOkIgnored() {
	assert := require.New(ts.T())
	buf := make([]byte, 65507)

	// register gateway
	p := packets.PullDataPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
	}
	b, err := p.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
	assert.NoError(err)
	i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
	assert.NoError(err)
	var ack packets.PullACKPacket
	assert.NoError(ack.UnmarshalBinary(buf[:i]))
	assert.Equal(p.RandomToken, ack.RandomToken)
	assert.Equal(p.ProtocolVersion, ack.ProtocolVersion)

	// set cache
	ts.backend.cache.Set("12345:frame", &gw.DownlinkFrame{
		GatewayId: "0102030405060708",
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
								SpreadingFactor:       7,
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
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
			},
			{
				PhyPayload: []byte{1, 2, 3, 4, 5},
				TxInfo: &gw.DownlinkTxInfo{
					Frequency: 868100000,
					Power:     14,
					Modulation: &gw.Modulation{
						Parameters: &gw.Modulation_Lora{
							Lora: &gw.LoraModulationInfo{
								Bandwidth:             125000,
								SpreadingFactor:       7,
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
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
			},
		},
		DownlinkId: 12345,
	}, cache.DefaultExpiration)
	ts.backend.cache.Set("12345:index", 0, cache.DefaultExpiration)
	ts.backend.cache.Set("12345:ack", []*gw.DownlinkTxAckItem{
		{Status: gw.TxAckStatus_IGNORED},
		{Status: gw.TxAckStatus_IGNORED},
	}, cache.DefaultExpiration)

	// send an ack on the first downlink attempt
	ack1 := packets.TXACKPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
		Payload: &packets.TXACKPayload{
			TXPKACK: packets.TXPKACK{
				Error: "",
			},
		},
	}
	ack1B, err := ack1.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(ack1B, ts.backendUDPAddr)
	assert.NoError(err)

	// validate final ack
	ackChan := make(chan *gw.DownlinkTxAck, 1)
	ts.backend.SetDownlinkTxAckFunc(func(pl *gw.DownlinkTxAck) {
		ackChan <- pl
	})
	txAck := <-ackChan
	assert.Equal(&gw.DownlinkTxAck{
		GatewayId:  "0102030405060708",
		DownlinkId: 12345,
		Items: []*gw.DownlinkTxAckItem{
			{
				Status: gw.TxAckStatus_OK,
			},
			{
				Status: gw.TxAckStatus_IGNORED,
			},
		},
	}, txAck)
}

func (ts *BackendTestSuite) TestPushData() {
	latitude := float64(1.234)
	longitude := float64(2.123)
	altitude := int32(123)

	now := time.Now().Round(time.Second)
	nowPB := timestamppb.New(now)

	compactTS := packets.CompactTime(now)
	tmms := int64(time.Second / time.Millisecond)

	testTable := []struct {
		Name          string
		GatewayPacket packets.PushDataPacket
		Stats         *gw.GatewayStats
		UplinkFrames  []*gw.UplinkFrame
	}{
		{
			Name: "stats with location",
			GatewayPacket: packets.PushDataPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     1234,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: packets.PushDataPayload{
					Stat: &packets.Stat{
						Time: packets.ExpandedTime(now.UTC()),
						Lati: latitude,
						Long: longitude,
						Alti: altitude,
						RXNb: 1,
						RXOK: 2,
						RXFW: 3,
						ACKR: 33.3,
						DWNb: 4,
						TXNb: 5,
					},
				},
			},
			Stats: &gw.GatewayStats{
				GatewayId: "0102030405060708",
				Time:      nowPB,
				Location: &common.Location{
					Latitude:  1.234,
					Longitude: 2.123,
					Altitude:  123,
					Source:    common.LocationSource_GPS,
				},
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 2,
				TxPacketsReceived:   4,
				TxPacketsEmitted:    5,
			},
		},
		{
			Name: "stats without location",
			GatewayPacket: packets.PushDataPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     1234,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: packets.PushDataPayload{
					Stat: &packets.Stat{
						Time: packets.ExpandedTime(now.UTC()),
						RXNb: 1,
						RXOK: 2,
						RXFW: 3,
						ACKR: 33.3,
						DWNb: 4,
						TXNb: 5,
					},
				},
			},
			Stats: &gw.GatewayStats{
				GatewayId:           "0102030405060708",
				Time:                nowPB,
				RxPacketsReceived:   1,
				RxPacketsReceivedOk: 2,
				TxPacketsReceived:   4,
				TxPacketsEmitted:    5,
			},
		},
		{
			Name: "rxpk",
			GatewayPacket: packets.PushDataPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     1234,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				Payload: packets.PushDataPayload{
					RXPK: []packets.RXPK{
						{
							Time: &compactTS,
							Tmst: 708016819,
							Tmms: &tmms,
							Freq: 868.5,
							Chan: 2,
							RFCh: 1,
							Stat: 1,
							Modu: "LORA",
							DatR: packets.DatR{LoRa: "SF7BW125"},
							CodR: "4/5",
							RSSI: -51,
							LSNR: 7,
							Size: 16,
							Data: []byte{64, 1, 1, 1, 1, 128, 0, 0, 1, 85, 247, 99, 71, 166, 43, 75},
						},
					},
				},
			},
			UplinkFrames: []*gw.UplinkFrame{
				{
					PhyPayload: []byte{64, 1, 1, 1, 1, 128, 0, 0, 1, 85, 247, 99, 71, 166, 43, 75},
					TxInfo: &gw.UplinkTxInfo{
						Frequency: 868500000,
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
						GatewayId:         "0102030405060708",
						GwTime:            nowPB,
						TimeSinceGpsEpoch: durationpb.New(time.Second),
						Rssi:              -51,
						Snr:               7,
						Channel:           2,
						RfChain:           1,
						Context:           []byte{0x2a, 0x33, 0x7a, 0xb3},
						CrcStatus:         gw.CRCStatus_CRC_OK,
					},
				},
			},
		},
	}

	for _, test := range testTable {
		ts.T().Run(test.Name, func(t *testing.T) {
			assert := require.New(t)

			statsChan := make(chan *gw.GatewayStats, 1)
			uplinkChan := make(chan *gw.UplinkFrame, 1)

			ts.backend.SetGatewayStatsFunc(func(pl *gw.GatewayStats) {
				statsChan <- pl
			})
			ts.backend.SetUplinkFrameFunc(func(pl *gw.UplinkFrame) {
				uplinkChan <- pl
			})

			// send gateway data
			b, err := test.GatewayPacket.MarshalBinary()
			assert.NoError(err)
			_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
			assert.NoError(err)

			// expect ack
			buf := make([]byte, 65507)
			i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
			assert.NoError(err)
			var ack packets.PushACKPacket
			assert.NoError(ack.UnmarshalBinary(buf[:i]))
			assert.Equal(test.GatewayPacket.RandomToken, ack.RandomToken)
			assert.Equal(test.GatewayPacket.ProtocolVersion, ack.ProtocolVersion)

			// stats
			if test.Stats != nil {
				stats := <-statsChan
				assert.True(proto.Equal(test.Stats, stats))
			}

			// uplink frames
			for _, uf := range test.UplinkFrames {
				receivedUF := <-uplinkChan

				assert.NotEqual(receivedUF.RxInfo.UplinkId, 0)
				receivedUF.RxInfo.UplinkId = 0

				assert.True(proto.Equal(uf, receivedUF))
			}
		})
	}
}

func (ts *BackendTestSuite) TestSendDownlinkFrame() {
	assert := require.New(ts.T())

	tmst := uint32(2000000)

	testTable := []struct {
		Name          string
		DownlinkFrame *gw.DownlinkFrame
		GatewayPacket packets.PullRespPacket
		Error         error
	}{
		{
			Name: "Gateway not registered",
			DownlinkFrame: &gw.DownlinkFrame{
				GatewayId: "0101010101010101",
				Items: []*gw.DownlinkFrameItem{
					{
						TxInfo: &gw.DownlinkTxInfo{},
					},
				},
			},
			Error: errors.New("get gateway error: gateway does not exist"),
		},
		{
			Name: "LORA",
			DownlinkFrame: &gw.DownlinkFrame{
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
										SpreadingFactor:       7,
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
							Board:   1,
							Antenna: 2,
							Context: []byte{0x00, 0x0f, 0x42, 0x40},
						},
					},
				},
				DownlinkId: 123,
				GatewayId:  "0102030405060708",
			},
			GatewayPacket: packets.PullRespPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     123,
				Payload: packets.PullRespPayload{
					TXPK: packets.TXPK{
						Tmst: &tmst,
						Freq: 868.1,
						RFCh: 0,
						Powe: 14,
						Modu: "LORA",
						DatR: packets.DatR{
							LoRa: "SF7BW125",
						},
						CodR: "4/5",
						IPol: true,
						Size: 4,
						Data: []byte{1, 2, 3, 4},
						Brd:  1,
						Ant:  2,
					},
				},
			},
		},
		{
			Name: "FSK",
			DownlinkFrame: &gw.DownlinkFrame{
				Items: []*gw.DownlinkFrameItem{
					{
						PhyPayload: []byte{1, 2, 3, 4},
						TxInfo: &gw.DownlinkTxInfo{
							Frequency: 868100000,
							Power:     14,
							Modulation: &gw.Modulation{
								Parameters: &gw.Modulation_Fsk{
									Fsk: &gw.FskModulationInfo{
										Datarate:           50000,
										FrequencyDeviation: 25000,
									},
								},
							},
							Board:   1,
							Antenna: 2,
							Timing: &gw.Timing{
								Parameters: &gw.Timing_Delay{
									Delay: &gw.DelayTimingInfo{
										Delay: durationpb.New(time.Second),
									},
								},
							},
							Context: []byte{0x00, 0x0f, 0x42, 0x40},
						},
					},
				},
				DownlinkId: 123,
				GatewayId:  "0102030405060708",
			},
			GatewayPacket: packets.PullRespPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     123,
				Payload: packets.PullRespPayload{
					TXPK: packets.TXPK{
						Tmst: &tmst,
						Freq: 868.1,
						RFCh: 0,
						Powe: 14,
						Modu: "FSK",
						FDev: 25000,
						DatR: packets.DatR{
							FSK: 50000,
						},
						Size: 4,
						Data: []byte{1, 2, 3, 4},
						Brd:  1,
						Ant:  2,
					},
				},
			},
		},
	}

	// register gateway
	p := packets.PullDataPacket{
		ProtocolVersion: packets.ProtocolVersion2,
		RandomToken:     12345,
		GatewayMAC:      lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
	}
	b, err := p.MarshalBinary()
	assert.NoError(err)
	_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
	assert.NoError(err)

	buf := make([]byte, 65507)
	i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
	assert.NoError(err)
	var ack packets.PullACKPacket
	assert.NoError(ack.UnmarshalBinary(buf[:i]))
	assert.Equal(p.RandomToken, ack.RandomToken)
	assert.Equal(p.ProtocolVersion, ack.ProtocolVersion)

	for _, test := range testTable {
		ts.T().Run(test.Name, func(t *testing.T) {
			assert := require.New(t)

			err := ts.backend.SendDownlinkFrame(test.DownlinkFrame)
			if test.Error != nil {
				assert.Error(err)
				assert.Equal(test.Error.Error(), err.Error())
				return
			}
			assert.NoError(err)

			i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
			assert.NoError(err)

			var pullResp packets.PullRespPacket
			assert.NoError(pullResp.UnmarshalBinary(buf[:i]))
			assert.Equal(test.GatewayPacket, pullResp)
		})
	}
}

func TestBackend(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
