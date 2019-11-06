package semtechudp

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/semtechudp/packets"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-api/go/common"
	"github.com/brocaar/chirpstack-api/go/gw"
	"github.com/brocaar/lorawan"
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
	conf.Backend.SemtechUDP.Configuration = []struct {
		GatewayID      string `mapstructure:"gateway_id"`
		BaseFile       string `mapstructure:"base_file"`
		OutputFile     string `mapstructure:"output_file"`
		RestartCommand string `mapstructure:"restart_command"`
	}{
		{
			GatewayID:      "0102030405060708",
			BaseFile:       filepath.Join("test/test.json"),
			OutputFile:     filepath.Join(ts.tempDir, "out.json"),
			RestartCommand: "touch " + filepath.Join(ts.tempDir, "restart"),
		},
	}

	ts.backend, err = NewBackend(conf)
	assert.NoError(err)

	ts.backendUDPAddr, err = net.ResolveUDPAddr("udp", ts.backend.conn.LocalAddr().String())
	assert.NoError(err)

	gwAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	assert.NoError(err)

	ts.gwUDPConn, err = net.ListenUDP("udp", gwAddr)
	assert.NoError(err)
	assert.NoError(ts.gwUDPConn.SetDeadline(time.Now().Add(time.Second)))

	go func() {
		for {
			<-ts.backend.GetConnectChan()
		}
	}()

	go func() {
		for {
			<-ts.backend.GetDisconnectChan()
		}
	}()
}

func (ts *BackendTestSuite) TearDownTest() {
	os.RemoveAll(ts.tempDir)
	ts.backend.Close()
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
		BackendPacket gw.DownlinkTXAck
	}{
		{
			Name: "no error",
			GatewayPacket: packets.TXACKPacket{
				ProtocolVersion: packets.ProtocolVersion2,
				RandomToken:     12345,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			},
			BackendPacket: gw.DownlinkTXAck{
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Token:     12345,
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
			BackendPacket: gw.DownlinkTXAck{
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Token:     12345,
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
						Error: "BOOM",
					},
				},
			},
			BackendPacket: gw.DownlinkTXAck{
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Token:     12345,
				Error:     "BOOM",
			},
		},
	}

	for _, test := range testTable {
		ts.T().Run(test.Name, func(t *testing.T) {
			assert := require.New(t)
			id, err := uuid.NewV4()
			assert.NoError(err)

			ts.backend.tokenMap[12345] = id[:]

			b, err := test.GatewayPacket.MarshalBinary()
			assert.NoError(err)
			_, err = ts.gwUDPConn.WriteToUDP(b, ts.backendUDPAddr)
			assert.NoError(err)

			ack := <-ts.backend.GetDownlinkTXAckChan()
			assert.Equal(id[:], ack.DownlinkId)
			ack.DownlinkId = nil

			assert.Equal(test.BackendPacket, ack)
		})
	}
}

func (ts *BackendTestSuite) TestPushData() {
	latitude := float64(1.234)
	longitude := float64(2.123)
	altitude := int32(123)

	now := time.Now().Round(time.Second)
	nowPB, _ := ptypes.TimestampProto(now)

	compactTS := packets.CompactTime(now)
	tmms := int64(time.Second / time.Millisecond)

	testTable := []struct {
		Name          string
		GatewayPacket packets.PushDataPacket
		Stats         *gw.GatewayStats
		UplinkFrames  []gw.UplinkFrame
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
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
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
				GatewayId:           []byte{1, 2, 3, 4, 5, 6, 7, 8},
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
			UplinkFrames: []gw.UplinkFrame{
				{
					PhyPayload: []byte{64, 1, 1, 1, 1, 128, 0, 0, 1, 85, 247, 99, 71, 166, 43, 75},
					TxInfo: &gw.UplinkTXInfo{
						Frequency:  868500000,
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
						GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
						Time:      nowPB,
						TimeSinceGpsEpoch: &duration.Duration{
							Seconds: 1,
						},
						Rssi:    -51,
						LoraSnr: 7,
						Channel: 2,
						RfChain: 1,
						Context: []byte{0x2a, 0x33, 0x7a, 0xb3},
					},
				},
			},
		},
	}

	for _, test := range testTable {
		ts.T().Run(test.Name, func(t *testing.T) {
			assert := require.New(t)

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
				stats := <-ts.backend.GetGatewayStatsChan()
				ip, err := getOutboundIP()
				assert.NoError(err)
				test.Stats.Ip = ip.String()

				assert.Len(stats.StatsId, 16)
				stats.StatsId = nil

				assert.Equal(test.Stats, &stats)
			}

			// uplink frames
			for _, uf := range test.UplinkFrames {
				receivedUF := <-ts.backend.GetUplinkFrameChan()

				assert.Len(receivedUF.RxInfo.UplinkId, 16)
				receivedUF.RxInfo.UplinkId = nil

				assert.Equal(uf, receivedUF)
			}
		})
	}
}

func (ts *BackendTestSuite) TestSendDownlinkFrame() {
	assert := require.New(ts.T())
	id, err := uuid.NewV4()
	assert.NoError(err)

	tmst := uint32(2000000)

	testTable := []struct {
		Name          string
		DownlinkFrame gw.DownlinkFrame
		GatewayPacket packets.PullRespPacket
		Error         error
	}{
		{
			Name: "Gateway not registered",
			DownlinkFrame: gw.DownlinkFrame{
				TxInfo: &gw.DownlinkTXInfo{
					GatewayId: []byte{1, 1, 1, 1, 1, 1, 1, 1},
				},
			},
			Error: errors.New("get gateway error: gateway does not exist"),
		},
		{
			Name: "LORA",
			DownlinkFrame: gw.DownlinkFrame{
				PhyPayload: []byte{1, 2, 3, 4},
				TxInfo: &gw.DownlinkTXInfo{
					GatewayId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
					Frequency:  868100000,
					Power:      14,
					Modulation: common.Modulation_LORA,
					ModulationInfo: &gw.DownlinkTXInfo_LoraModulationInfo{
						LoraModulationInfo: &gw.LoRaModulationInfo{
							Bandwidth:             125,
							SpreadingFactor:       7,
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
					Board:   1,
					Antenna: 2,
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
				Token:      123,
				DownlinkId: id[:],
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
			DownlinkFrame: gw.DownlinkFrame{
				PhyPayload: []byte{1, 2, 3, 4},
				TxInfo: &gw.DownlinkTXInfo{
					GatewayId:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
					Frequency:  868100000,
					Power:      14,
					Modulation: common.Modulation_FSK,
					ModulationInfo: &gw.DownlinkTXInfo_FskModulationInfo{
						FskModulationInfo: &gw.FSKModulationInfo{
							Bitrate: 50000,
						},
					},
					Board:   1,
					Antenna: 2,
					Timing:  gw.DownlinkTiming_DELAY,
					TimingInfo: &gw.DownlinkTXInfo_DelayTimingInfo{
						DelayTimingInfo: &gw.DelayTimingInfo{
							Delay: ptypes.DurationProto(time.Second),
						},
					},
					Context: []byte{0x00, 0x0f, 0x42, 0x40},
				},
				Token:      123,
				DownlinkId: id[:],
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

			assert.Equal(id[:], ts.backend.tokenMap[uint16(test.DownlinkFrame.Token)])

			i, _, err := ts.gwUDPConn.ReadFromUDP(buf)
			assert.NoError(err)

			var pullResp packets.PullRespPacket
			assert.NoError(pullResp.UnmarshalBinary(buf[:i]))
			assert.Equal(test.GatewayPacket, pullResp)
		})
	}
}

func (ts *BackendTestSuite) TestApplyConfiguration() {
	testTable := []struct {
		Name                    string
		GatewayConfiguration    gw.GatewayConfiguration
		ExpectedRadios          [2]map[string]interface{}
		ExpectedLoRaStd         map[string]interface{}
		ExpectedFSK             map[string]interface{}
		ExpectedMultiSFChannels []map[string]interface{}
	}{

		{
			Name: "EU 868 band config (minimal configuration)",
			GatewayConfiguration: gw.GatewayConfiguration{
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Channels: []*gw.ChannelConfiguration{
					{
						Frequency:  868100000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  868300000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  868500000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
				},
			},
			ExpectedRadios: [2]map[string]interface{}{
				{
					"enable": true,
					"freq":   868500000,
				},
				{
					"enable": false,
				},
			},
			ExpectedLoRaStd: map[string]interface{}{
				"enable": false,
			},
			ExpectedFSK: map[string]interface{}{
				"enable": false,
			},
			ExpectedMultiSFChannels: []map[string]interface{}{
				{
					"enable": true,
					"if":     -400000,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     -200000,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     0,
					"radio":  0,
				},
				{
					"enable": false,
				},
			},
		},
		{
			Name: "EU 868 band config + CFList + LoRa single-SF + FSK",
			GatewayConfiguration: gw.GatewayConfiguration{
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Channels: []*gw.ChannelConfiguration{
					{
						Frequency:  868100000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  868300000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  868500000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  867100000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  867300000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  867500000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  867700000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  867900000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10, 11, 12},
							},
						},
					},
					{
						Frequency:  868300000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        250,
								SpreadingFactors: []uint32{7},
							},
						},
					},
					{
						Frequency:  868800000,
						Modulation: common.Modulation_FSK,
						ModulationConfig: &gw.ChannelConfiguration_FskModulationConfig{
							FskModulationConfig: &gw.FSKModulationConfig{
								Bandwidth: 125,
								Bitrate:   50000,
							},
						},
					},
				},
			},
			ExpectedRadios: [2]map[string]interface{}{
				{
					"enable": true,
					"freq":   867500000,
				},
				{
					"enable": true,
					"freq":   868500000,
				},
			},
			ExpectedLoRaStd: map[string]interface{}{
				"bandwidth":     250000,
				"enable":        true,
				"if":            -200000,
				"radio":         1,
				"spread_factor": 7,
			},
			ExpectedFSK: map[string]interface{}{
				"bandwidth": 125,
				"datarate":  50000,
				"enable":    true,
				"if":        300000,
				"radio":     1,
			},
			ExpectedMultiSFChannels: []map[string]interface{}{
				{
					"enable": true,
					"if":     -400000,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     -200000,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     0,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     200000,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     400000,
					"radio":  0,
				},
				{
					"enable": true,
					"if":     -400000,
					"radio":  1,
				},
				{
					"enable": true,
					"if":     -200000,
					"radio":  1,
				},
				{
					"enable": true,
					"if":     0,
					"radio":  1,
				},
			},
		},
		{
			Name: "US band (0-7 + 64)",
			GatewayConfiguration: gw.GatewayConfiguration{
				GatewayId: []byte{1, 2, 3, 4, 5, 6, 7, 8},
				Channels: []*gw.ChannelConfiguration{
					{
						Frequency:  902300000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  902500000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  902700000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  902900000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  903100000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  903300000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  903500000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  903700000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        125,
								SpreadingFactors: []uint32{7, 8, 9, 10},
							},
						},
					},
					{
						Frequency:  903000000,
						Modulation: common.Modulation_LORA,
						ModulationConfig: &gw.ChannelConfiguration_LoraModulationConfig{
							LoraModulationConfig: &gw.LoRaModulationConfig{
								Bandwidth:        500,
								SpreadingFactors: []uint32{8},
							},
						},
					},
				},
			},
			ExpectedRadios: [2]map[string]interface{}{
				{
					"enable": true,
					"freq":   902700000,
				},
				{
					"enable": true,
					"freq":   903700000,
				},
			},
			ExpectedLoRaStd: map[string]interface{}{
				"enable":        true,
				"radio":         0,
				"if":            300000,
				"bandwidth":     500000,
				"spread_factor": 8,
			},
			ExpectedFSK: map[string]interface{}{
				"enable": false,
			},
			ExpectedMultiSFChannels: []map[string]interface{}{
				{
					"enable": true,
					"radio":  0,
					"if":     -400000,
				},
				{
					"enable": true,
					"radio":  0,
					"if":     -200000,
				},
				{
					"enable": true,
					"radio":  0,
					"if":     0,
				},
				{
					"enable": true,
					"radio":  0,
					"if":     200000,
				},
				{
					"enable": true,
					"radio":  0,
					"if":     400000,
				},
				{
					"enable": true,
					"radio":  1,
					"if":     -400000,
				},
				{
					"enable": true,
					"radio":  1,
					"if":     -200000,
				},
				{
					"enable": true,
					"radio":  1,
					"if":     0,
				},
			},
		},
	}

	for _, test := range testTable {
		ts.T().Run(test.Name, func(t *testing.T) {
			assert := require.New(t)

			err := ts.backend.ApplyConfiguration(test.GatewayConfiguration)
			assert.NoError(err)

			if len(test.ExpectedRadios) == 0 {
				// pf has not been restarted
				_, err = os.Stat(filepath.Join(ts.tempDir, "restart"))
				assert.Error(err)
				return
			}

			// pf has been restarted
			_, err = os.Stat(filepath.Join(ts.tempDir, "restart"))
			assert.NoError(err)

			// new config has been written
			pfConfig, err := loadConfigFile(filepath.Join(ts.tempDir, "out.json"))
			assert.NoError(err)

			// radios are configured as expected
			for i, expConf := range test.ExpectedRadios {
				radio := pfConfig.SX1301Conf[fmt.Sprintf("radio_%d", i)].(map[string]interface{})
				for k, v := range expConf {
					assert.EqualValues(v, radio[k])
				}
			}

			// LoRa Std
			loRaChann := pfConfig.SX1301Conf["chan_Lora_std"].(map[string]interface{})
			for k, v := range test.ExpectedLoRaStd {
				assert.EqualValues(v, loRaChann[k])
			}

			// FSK
			fskChan := pfConfig.SX1301Conf["chan_FSK"].(map[string]interface{})
			for k, v := range test.ExpectedFSK {
				assert.EqualValues(v, fskChan[k])
			}

			// Multi-SF LoRa
			for i, expConf := range test.ExpectedMultiSFChannels {
				channel, ok := pfConfig.SX1301Conf[fmt.Sprintf("chan_multiSF_%d", i)].(map[string]interface{})
				assert.True(ok)
				for k, v := range expConf {
					assert.EqualValues(v, channel[k])
				}
			}
		})
	}
}

func TestBackend(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
