package gateway

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBackend(t *testing.T) {
	Convey("Given a new Backend binding at a random port", t, func() {
		tempDir, err := ioutil.TempDir("", "test")
		So(err, ShouldBeNil)
		defer os.RemoveAll(tempDir)

		backend, err := NewBackend("127.0.0.1:0", nil, nil, false, []Configuration{
			{
				MAC:            lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
				BaseFile:       filepath.Join("test/test.json"),
				OutputFile:     filepath.Join(tempDir, "out.json"),
				RestartCommand: "touch " + filepath.Join(tempDir, "restart"),
				version:        "12345",
			},
		})
		So(err, ShouldBeNil)

		backendAddr, err := net.ResolveUDPAddr("udp", backend.conn.LocalAddr().String())
		So(err, ShouldBeNil)

		latitude := float64(1.234)
		longitude := float64(2.123)
		altitude := int32(123)

		Convey("Given a fake gateway UDP publisher", func() {
			gwAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
			So(err, ShouldBeNil)
			gwConn, err := net.ListenUDP("udp", gwAddr)
			So(err, ShouldBeNil)
			defer gwConn.Close()
			So(gwConn.SetDeadline(time.Now().Add(time.Second)), ShouldBeNil)

			Convey("When sending a PULL_DATA packet", func() {
				p := PullDataPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     12345,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				}
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				_, err = gwConn.WriteToUDP(b, backendAddr)
				So(err, ShouldBeNil)

				Convey("Then an ACK packet is returned", func() {
					buf := make([]byte, 65507)
					i, _, err := gwConn.ReadFromUDP(buf)
					So(err, ShouldBeNil)
					var ack PullACKPacket
					So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
					So(ack.RandomToken, ShouldEqual, p.RandomToken)
					So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)
				})
			})

			Convey("When sending a TX_ACK packet (no error)", func() {
				p := TXACKPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     12345,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				}
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				_, err = gwConn.WriteToUDP(b, backendAddr)
				So(err, ShouldBeNil)

				Convey("Then the ack is returned by the ack channel", func() {
					ack := <-backend.TXAckChan()
					So(ack, ShouldResemble, gw.TXAck{MAC: p.GatewayMAC, Token: p.RandomToken})
				})
			})

			Convey("When sending a TX_ACK packet (with error)", func() {
				p := TXACKPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     12345,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Payload: &TXACKPayload{
						TXPKACK: TXPKACK{
							Error: gw.ErrGPSUnlocked,
						},
					},
				}
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				_, err = gwConn.WriteToUDP(b, backendAddr)
				So(err, ShouldBeNil)

				Convey("Then the ack is returned by the ack channel", func() {
					ack := <-backend.TXAckChan()
					So(ack, ShouldResemble, gw.TXAck{MAC: p.GatewayMAC, Token: p.RandomToken, Error: gw.ErrGPSUnlocked})
				})
			})

			Convey("When sending a TX_ACK packet (with error 'NONE')", func() {
				p := TXACKPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     12345,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Payload: &TXACKPayload{
						TXPKACK: TXPKACK{
							Error: "NONE",
						},
					},
				}
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				_, err = gwConn.WriteToUDP(b, backendAddr)
				So(err, ShouldBeNil)

				Convey("Then the ack is returned by the ack channel", func() {
					ack := <-backend.TXAckChan()
					So(ack, ShouldResemble, gw.TXAck{MAC: p.GatewayMAC, Token: p.RandomToken})
				})
			})

			Convey("When sending a PUSH_DATA packet with stats", func() {
				p := PushDataPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     1234,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Payload: PushDataPayload{
						Stat: &Stat{
							Time: ExpandedTime(time.Time{}.UTC()),
							Lati: &latitude,
							Long: &longitude,
							Alti: &altitude,
							RXNb: 1,
							RXOK: 2,
							RXFW: 3,
							ACKR: 33.3,
							DWNb: 4,
							TXNb: 3,
						},
					},
				}
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				_, err = gwConn.WriteToUDP(b, backendAddr)
				So(err, ShouldBeNil)

				Convey("Then an ACK packet is returned", func() {
					buf := make([]byte, 65507)
					i, _, err := gwConn.ReadFromUDP(buf)
					So(err, ShouldBeNil)
					var ack PushACKPacket
					So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
					So(ack.RandomToken, ShouldEqual, p.RandomToken)
					So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)

					Convey("Then the gateway stats are returned by the stats channel", func() {
						stats := <-backend.StatsChan()
						So([8]byte(stats.MAC), ShouldEqual, [8]byte{1, 2, 3, 4, 5, 6, 7, 8})
						So(stats.ConfigVersion, ShouldEqual, "12345")
					})
				})
			})

			Convey("Given skipCRCCheck=false", func() {
				backend.skipCRCCheck = false

				Convey("When sending a PUSH_DATA packet with RXPK (CRC OK + GPS timestamp)", func() {
					ts := CompactTime(time.Now().UTC())
					tmms := int64(time.Second / time.Millisecond)

					p := PushDataPacket{
						ProtocolVersion: ProtocolVersion2,
						RandomToken:     1234,
						GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
						Payload: PushDataPayload{
							RXPK: []RXPK{
								{
									Time: &ts,
									Tmst: 708016819,
									Tmms: &tmms,
									Freq: 868.5,
									Chan: 2,
									RFCh: 1,
									Stat: 1,
									Modu: "LORA",
									DatR: DatR{LoRa: "SF7BW125"},
									CodR: "4/5",
									RSSI: -51,
									LSNR: 7,
									Size: 16,
									Data: "QAEBAQGAAAABVfdjR6YrSw==",
								},
							},
						},
					}
					b, err := p.MarshalBinary()
					So(err, ShouldBeNil)
					_, err = gwConn.WriteToUDP(b, backendAddr)
					So(err, ShouldBeNil)

					Convey("Then an ACK packet is returned", func() {
						buf := make([]byte, 65507)
						i, _, err := gwConn.ReadFromUDP(buf)
						So(err, ShouldBeNil)
						var ack PushACKPacket
						So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
						So(ack.RandomToken, ShouldEqual, p.RandomToken)
						So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)
					})

					Convey("Then the packet is returned by the RX packet channel", func() {
						rxPacket := <-backend.RXPacketChan()

						rxPackets, err := newRXPacketsFromRXPK(p.GatewayMAC, p.Payload.RXPK[0])
						So(err, ShouldBeNil)
						So(rxPacket, ShouldResemble, rxPackets[0])
					})
				})

				Convey("When sending a PUSH_DATA packet with RXPK (CRC not OK)", func() {
					p := PushDataPacket{
						ProtocolVersion: ProtocolVersion2,
						RandomToken:     1234,
						GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
						Payload: PushDataPayload{
							RXPK: []RXPK{
								{
									Tmst: 708016819,
									Freq: 868.5,
									Chan: 2,
									RFCh: 1,
									Stat: -1,
									Modu: "LORA",
									DatR: DatR{LoRa: "SF7BW125"},
									CodR: "4/5",
									RSSI: -51,
									LSNR: 7,
									Size: 16,
									Data: "QAEBAQGAAAABVfdjR6YrSw==",
								},
							},
						},
					}
					b, err := p.MarshalBinary()
					So(err, ShouldBeNil)
					_, err = gwConn.WriteToUDP(b, backendAddr)
					So(err, ShouldBeNil)

					Convey("Then an ACK packet is returned", func() {
						buf := make([]byte, 65507)
						i, _, err := gwConn.ReadFromUDP(buf)
						So(err, ShouldBeNil)
						var ack PushACKPacket
						So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
						So(ack.RandomToken, ShouldEqual, p.RandomToken)
						So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)
					})

					Convey("Then the packet is not returned by the RX packet channel", func() {
						So(backend.RXPacketChan(), ShouldHaveLength, 0)
					})
				})
			})

			Convey("Given skipCRCCheck=true", func() {
				backend.skipCRCCheck = true

				Convey("When sending a PUSH_DATA packet with RXPK (CRC OK + GPS timestamp)", func() {
					ts := CompactTime(time.Now().UTC())
					p := PushDataPacket{
						ProtocolVersion: ProtocolVersion2,
						RandomToken:     1234,
						GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
						Payload: PushDataPayload{
							RXPK: []RXPK{
								{
									Time: &ts,
									Tmst: 708016819,
									Freq: 868.5,
									Chan: 2,
									RFCh: 1,
									Stat: 1,
									Modu: "LORA",
									DatR: DatR{LoRa: "SF7BW125"},
									CodR: "4/5",
									RSSI: -51,
									LSNR: 7,
									Size: 16,
									Data: "QAEBAQGAAAABVfdjR6YrSw==",
								},
							},
						},
					}
					b, err := p.MarshalBinary()
					So(err, ShouldBeNil)
					_, err = gwConn.WriteToUDP(b, backendAddr)
					So(err, ShouldBeNil)

					Convey("Then an ACK packet is returned", func() {
						buf := make([]byte, 65507)
						i, _, err := gwConn.ReadFromUDP(buf)
						So(err, ShouldBeNil)
						var ack PushACKPacket
						So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
						So(ack.RandomToken, ShouldEqual, p.RandomToken)
						So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)
					})

					Convey("Then the packet is returned by the RX packet channel", func() {
						rxPacket := <-backend.RXPacketChan()

						rxPackets, err := newRXPacketsFromRXPK(p.GatewayMAC, p.Payload.RXPK[0])
						So(err, ShouldBeNil)
						So(rxPacket, ShouldResemble, rxPackets[0])
					})
				})

				Convey("When sending a PUSH_DATA packet with RXPK (CRC not OK)", func() {
					p := PushDataPacket{
						ProtocolVersion: ProtocolVersion2,
						RandomToken:     1234,
						GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
						Payload: PushDataPayload{
							RXPK: []RXPK{
								{
									Tmst: 708016819,
									Freq: 868.5,
									Chan: 2,
									RFCh: 1,
									Stat: -1,
									Modu: "LORA",
									DatR: DatR{LoRa: "SF7BW125"},
									CodR: "4/5",
									RSSI: -51,
									LSNR: 7,
									Size: 16,
									Data: "QAEBAQGAAAABVfdjR6YrSw==",
								},
							},
						},
					}
					b, err := p.MarshalBinary()
					So(err, ShouldBeNil)
					_, err = gwConn.WriteToUDP(b, backendAddr)
					So(err, ShouldBeNil)

					Convey("Then an ACK packet is returned", func() {
						buf := make([]byte, 65507)
						i, _, err := gwConn.ReadFromUDP(buf)
						So(err, ShouldBeNil)
						var ack PushACKPacket
						So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
						So(ack.RandomToken, ShouldEqual, p.RandomToken)
						So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)
					})

					Convey("Then the packet is returned by the RX packet channel", func() {
						rxPacket := <-backend.RXPacketChan()

						rxPackets, err := newRXPacketsFromRXPK(p.GatewayMAC, p.Payload.RXPK[0])
						So(err, ShouldBeNil)
						So(rxPacket, ShouldResemble, rxPackets[0])
					})
				})
			})

			Convey("Given a TXPacket", func() {
				internalTS := uint32(12345)
				timeSinceGPSEpoch := gw.Duration(time.Second)

				txPacket := gw.TXPacketBytes{
					TXInfo: gw.TXInfo{
						MAC:               [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
						Immediately:       true,
						Timestamp:         &internalTS,
						TimeSinceGPSEpoch: &timeSinceGPSEpoch,
						Frequency:         868100000,
						Power:             14,
						DataRate: band.DataRate{
							Modulation:   band.LoRaModulation,
							SpreadFactor: 12,
							Bandwidth:    250,
						},
						CodeRate: "4/5",
					},
					PHYPayload: []byte{1, 2, 3, 4},
				}

				Convey("When sending the TXPacket and the gateway is not known to the backend", func() {
					err := backend.Send(txPacket)
					Convey("Then the backend returns an error", func() {
						So(err, ShouldEqual, errGatewayDoesNotExist)
					})
				})

				Convey("When sending the TXPacket when the gateway is known to the backend", func() {
					// sending a ping should register the gateway to the backend
					p := PullDataPacket{
						ProtocolVersion: ProtocolVersion2,
						RandomToken:     12345,
						GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					}
					b, err := p.MarshalBinary()
					So(err, ShouldBeNil)
					_, err = gwConn.WriteToUDP(b, backendAddr)
					So(err, ShouldBeNil)
					buf := make([]byte, 65507)
					i, _, err := gwConn.ReadFromUDP(buf)
					So(err, ShouldBeNil)
					var ack PullACKPacket
					So(ack.UnmarshalBinary(buf[:i]), ShouldBeNil)
					So(ack.RandomToken, ShouldEqual, p.RandomToken)
					So(ack.ProtocolVersion, ShouldEqual, p.ProtocolVersion)

					err = backend.Send(txPacket)

					Convey("Then no error is returned", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the data is received by the gateway", func() {
						i, _, err := gwConn.ReadFromUDP(buf)
						So(err, ShouldBeNil)
						So(i, ShouldBeGreaterThan, 0)
						var pullResp PullRespPacket
						So(pullResp.UnmarshalBinary(buf[:i]), ShouldBeNil)

						tmms := int64(time.Second / time.Millisecond)
						So(pullResp, ShouldResemble, PullRespPacket{
							ProtocolVersion: p.ProtocolVersion,
							Payload: PullRespPayload{
								TXPK: TXPK{
									Imme: true,
									Tmst: &internalTS,
									Tmms: &tmms,
									Freq: 868.1,
									Powe: 14,
									Modu: "LORA",
									DatR: DatR{
										LoRa: "SF12BW250",
									},
									CodR: "4/5",
									Size: uint16(len([]byte{1, 2, 3, 4})),
									Data: base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}),
									IPol: true,
								},
							},
						})
					})
				})
			})
		})

		Convey("Given a set of configuration tests", func() {
			testTable := []struct {
				Name                    string
				ConfigurationPacket     gw.GatewayConfigPacket
				ExpectedRadios          [2]map[string]interface{}
				ExpectedLoRaStd         map[string]interface{}
				ExpectedFSK             map[string]interface{}
				ExpectedMultiSFChannels []map[string]interface{}
			}{
				{
					Name: "EU 868 band config (minimal configuration)",
					ConfigurationPacket: gw.GatewayConfigPacket{
						MAC: lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
						Channels: []gw.Channel{
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868100000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868300000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868500000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
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
					ConfigurationPacket: gw.GatewayConfigPacket{
						MAC: lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
						Channels: []gw.Channel{
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868100000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868300000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868500000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        867100000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        867300000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        867500000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        867700000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        867900000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10, 11, 12},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        868300000,
								Bandwidth:        250,
								SpreadingFactors: []int{7},
							},
							{
								Modulation: band.FSKModulation,
								Frequency:  868800000,
								Bandwidth:  125,
								Bitrate:    50000,
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
					ConfigurationPacket: gw.GatewayConfigPacket{
						MAC: lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8},
						Channels: []gw.Channel{
							{
								Modulation:       band.LoRaModulation,
								Frequency:        902300000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        902500000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        902700000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        902900000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        903100000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        903300000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        903500000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        903700000,
								Bandwidth:        125,
								SpreadingFactors: []int{7, 8, 9, 10},
							},
							{
								Modulation:       band.LoRaModulation,
								Frequency:        903000000,
								Bandwidth:        500,
								SpreadingFactors: []int{8},
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

			for i, test := range testTable {
				Convey(fmt.Sprintf("Testing: %s [%d]", test.Name, i), func() {
					So(backend.ApplyConfiguration(test.ConfigurationPacket), ShouldBeNil)

					if len(test.ExpectedRadios) == 0 {
						Convey("Then the packet-forwarder restart command has not been invoked", func() {
							_, err := os.Stat(filepath.Join(tempDir, "restart"))
							So(err, ShouldNotBeNil)
						})
						return
					}

					Convey("Then the packet-forwarder restart command has been invokend", func() {
						_, err := os.Stat(filepath.Join(tempDir, "restart"))
						So(err, ShouldBeNil)
					})

					Convey("Then the new configuration has been written", func() {
						pfConfig, err := loadConfigFile(filepath.Join(tempDir, "out.json"))
						So(err, ShouldBeNil)

						Convey("Then the radios are configured as expected", func() {
							for i, expectedConfig := range test.ExpectedRadios {
								radio := pfConfig.SX1301Conf[fmt.Sprintf("radio_%d", i)].(map[string]interface{})
								for k, v := range expectedConfig {
									So(radio[k], ShouldEqual, v)
								}
							}
						})

						Convey("Then the LoRa Std channel is configured as expected", func() {
							channel := pfConfig.SX1301Conf["chan_Lora_std"].(map[string]interface{})
							for k, v := range test.ExpectedLoRaStd {
								So(channel[k], ShouldEqual, v)
							}
						})

						Convey("Then the FSK channel is configured as expected", func() {
							channel := pfConfig.SX1301Conf["chan_FSK"].(map[string]interface{})
							for k, v := range test.ExpectedFSK {
								So(channel[k], ShouldEqual, v)
							}
						})

						Convey("Then the multi-sf LoRa channels are configured as expected", func() {
							for i, expectedConfig := range test.ExpectedMultiSFChannels {
								channel, ok := pfConfig.SX1301Conf[fmt.Sprintf("chan_multiSF_%d", i)].(map[string]interface{})
								So(ok, ShouldBeTrue)
								for k, v := range expectedConfig {
									So(channel[k], ShouldEqual, v)
								}
							}
						})
					})
				})
			}
		})
	})
}
