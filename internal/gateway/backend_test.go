package gateway

import (
	"encoding/base64"
	"net"
	"testing"
	"time"

	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	. "github.com/smartystreets/goconvey/convey"
)

func TestBackend(t *testing.T) {
	Convey("Given a new Backend binding at a random port", t, func() {
		backend, err := NewBackend("127.0.0.1:0", nil, nil, false)
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
	})
}

func TestNewGatewayStatPacket(t *testing.T) {
	Convey("Given a (Semtech) Stat struct and gateway MAC with GPS data", t, func() {
		latitude := float64(1.234)
		longitude := float64(2.123)
		altitude := int32(123)

		now := time.Now().UTC()
		stat := Stat{
			Time: ExpandedTime(now),
			Lati: &latitude,
			Long: &longitude,
			Alti: &altitude,
			RXNb: 1,
			RXOK: 2,
			RXFW: 3,
			ACKR: 33.3,
			DWNb: 4,
			TXNb: 3,
		}
		mac := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

		Convey("When calling newGatewayStatsPacket", func() {
			latitude := float64(1.234)
			longitude := float64(2.123)
			altitude := float64(123)

			gwStats := newGatewayStatsPacket(mac, stat)
			Convey("Then all fields are set correctly", func() {
				So(gwStats, ShouldResemble, gw.GatewayStatsPacket{
					Time:                now,
					MAC:                 mac,
					Latitude:            &latitude,
					Longitude:           &longitude,
					Altitude:            &altitude,
					RXPacketsReceived:   1,
					RXPacketsReceivedOK: 2,
					TXPacketsReceived:   4,
					TXPacketsEmitted:    3,
				})
			})
		})
	})

	Convey("Given a (Semtech) Stat struct and gateway MAC without GPS data", t, func() {
		now := time.Now().UTC()
		stat := Stat{
			Time: ExpandedTime(now),
			RXNb: 1,
			RXOK: 2,
			RXFW: 3,
			ACKR: 33.3,
			DWNb: 4,
			TXNb: 3,
		}
		mac := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

		Convey("When calling newGatewayStatsPacket", func() {
			gwStats := newGatewayStatsPacket(mac, stat)
			Convey("Then all fields are set correctly", func() {
				So(gwStats, ShouldResemble, gw.GatewayStatsPacket{
					Time:                now,
					MAC:                 mac,
					RXPacketsReceived:   1,
					RXPacketsReceivedOK: 2,
					TXPacketsReceived:   4,
					TXPacketsEmitted:    3,
				})
			})
		})
	})
}

func TestNewTXPKFromTXPacket(t *testing.T) {
	internalTS := uint32(12345)

	Convey("Given a TXPacket", t, func() {
		timeSinceGPSEpoch := gw.Duration(time.Second)

		txPacket := gw.TXPacketBytes{
			TXInfo: gw.TXInfo{
				Timestamp:         &internalTS,
				TimeSinceGPSEpoch: &timeSinceGPSEpoch,
				Frequency:         868100000,
				Power:             14,
				CodeRate:          "4/5",
				DataRate: band.DataRate{
					Modulation:   band.LoRaModulation,
					SpreadFactor: 9,
					Bandwidth:    250,
				},
				Board:   1,
				Antenna: 2,
			},
			PHYPayload: []byte{1, 2, 3, 4},
		}

		Convey("Then te expected TXPK is returned (with default IPol", func() {
			tmms := int64(time.Second / time.Millisecond)
			txpk, err := newTXPKFromTXPacket(txPacket)
			So(err, ShouldBeNil)
			So(txpk, ShouldResemble, TXPK{
				Imme: false,
				Tmst: &internalTS,
				Tmms: &tmms,
				Freq: 868.1,
				Powe: 14,
				Modu: "LORA",
				DatR: DatR{
					LoRa: "SF9BW250",
				},
				CodR: "4/5",
				Size: 4,
				Data: "AQIDBA==",
				IPol: true,
				Brd:  1,
				Ant:  2,
			})
		})

		Convey("Given IPol is requested to false", func() {
			f := false
			txPacket.TXInfo.IPol = &f

			Convey("Then the TXPK IPol is set to false", func() {
				txpk, err := newTXPKFromTXPacket(txPacket)
				So(err, ShouldBeNil)
				So(txpk.IPol, ShouldBeFalse)
			})
		})
	})
}

func TestNewRXPacketFromRXPK(t *testing.T) {
	Convey("Given a RXPK and gateway MAC", t, func() {
		now := time.Now().UTC()
		nowCompact := CompactTime(now)
		tmms := int64(time.Second / time.Millisecond)
		timeSinceGPSEpoch := gw.Duration(time.Second)

		rxpk := RXPK{
			Time: &nowCompact,
			Tmms: &tmms,
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
			Data: base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}),
		}
		mac := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

		Convey("When calling newRXPacketsFromRXPK without RSig field", func() {
			rxPackets, err := newRXPacketsFromRXPK(mac, rxpk)
			So(err, ShouldBeNil)
			So(rxPackets, ShouldHaveLength, 1)

			Convey("Then all fields are set correctly", func() {
				So(rxPackets[0].PHYPayload, ShouldResemble, []byte{1, 2, 3, 4})

				So(rxPackets[0].RXInfo, ShouldResemble, gw.RXInfo{
					MAC:               mac,
					Time:              &now,
					TimeSinceGPSEpoch: &timeSinceGPSEpoch,
					Timestamp:         708016819,
					Frequency:         868500000,
					Channel:           2,
					RFChain:           1,
					CRCStatus:         1,
					DataRate: band.DataRate{
						Modulation:   band.LoRaModulation,
						SpreadFactor: 7,
						Bandwidth:    125,
					},
					CodeRate: "4/5",
					RSSI:     -51,
					LoRaSNR:  7,
					Size:     16,
				})
			})
		})

		Convey("When calling newRXPacketsFromRXPK with multiple RSig elements", func() {
			rxpk.Brd = 2
			rxpk.RSig = []RSig{
				{
					Ant:   1,
					Chan:  3,
					LSNR:  1.5,
					RSSIC: -50,
				},
				{
					Ant:   2,
					Chan:  3,
					LSNR:  2,
					RSSIC: -30,
				},
			}
			rxPackets, err := newRXPacketsFromRXPK(mac, rxpk)
			So(err, ShouldBeNil)
			So(rxPackets, ShouldHaveLength, 2)

			Convey("Then all fields are set correctly", func() {
				So(rxPackets[0].PHYPayload, ShouldResemble, []byte{1, 2, 3, 4})
				So(rxPackets[0].RXInfo, ShouldResemble, gw.RXInfo{
					MAC:               mac,
					Time:              &now,
					TimeSinceGPSEpoch: &timeSinceGPSEpoch,
					Timestamp:         708016819,
					Frequency:         868500000,
					Channel:           3,
					RFChain:           1,
					CRCStatus:         1,
					DataRate: band.DataRate{
						Modulation:   band.LoRaModulation,
						SpreadFactor: 7,
						Bandwidth:    125,
					},
					CodeRate: "4/5",
					RSSI:     -50,
					LoRaSNR:  1.5,
					Size:     16,
					Antenna:  1,
					Board:    2,
				})

				So(rxPackets[1].PHYPayload, ShouldResemble, []byte{1, 2, 3, 4})
				So(rxPackets[1].RXInfo, ShouldResemble, gw.RXInfo{
					MAC:               mac,
					Time:              &now,
					TimeSinceGPSEpoch: &timeSinceGPSEpoch,
					Timestamp:         708016819,
					Frequency:         868500000,
					Channel:           3,
					RFChain:           1,
					CRCStatus:         1,
					DataRate: band.DataRate{
						Modulation:   band.LoRaModulation,
						SpreadFactor: 7,
						Bandwidth:    125,
					},
					CodeRate: "4/5",
					RSSI:     -30,
					LoRaSNR:  2,
					Size:     16,
					Antenna:  2,
					Board:    2,
				})
			})
		})
	})
}

func TestGatewaysCallbacks(t *testing.T) {
	Convey("Given a new gateways registry", t, func() {
		gw := gateways{
			gateways: make(map[lorawan.EUI64]gateway),
		}

		mac := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

		Convey("Given a onNew and onDelete callback", func() {
			var onNewCalls int
			var onDeleteCalls int

			gw.onNew = func(mac lorawan.EUI64) error {
				onNewCalls = onNewCalls + 1
				return nil
			}

			gw.onDelete = func(mac lorawan.EUI64) error {
				onDeleteCalls = onDeleteCalls + 1
				return nil
			}

			Convey("When adding a new gateway", func() {
				So(gw.set(mac, gateway{}), ShouldBeNil)

				Convey("Then onNew callback is called once", func() {
					So(onNewCalls, ShouldEqual, 1)
				})

				Convey("When updating the same gateway", func() {
					So(gw.set(mac, gateway{}), ShouldBeNil)

					Convey("Then onNew has not been called", func() {
						So(onNewCalls, ShouldEqual, 1)
					})
				})

				Convey("When cleaning up the gateways", func() {
					So(gw.cleanup(), ShouldBeNil)

					Convey("Then onDelete has been called once", func() {
						So(onDeleteCalls, ShouldEqual, 1)
					})
				})
			})
		})
	})
}

func TestNewTXAckFromTXPKACK(t *testing.T) {
	Convey("Given a TXPKACK with error and a gateway mac", t, func() {
		mac := lorawan.EUI64{1, 2, 3, 4, 5, 6, 7, 8}
		ack := TXPKACK{
			Error: gw.ErrTooEarly,
		}

		Convey("Then newTXAckFromTXPKACK returns the expected value", func() {
			txAck := newTXAckFromTXPKACK(mac, 12345, ack)
			So(txAck, ShouldResemble, gw.TXAck{
				MAC:   mac,
				Token: 12345,
				Error: gw.ErrTooEarly,
			})
		})

		Convey("Given the TXPKACK does not contain an error", func() {
			ack.Error = "NONE"

			Convey("Then newTXAckFromTXPKACK returns the expected value", func() {
				txAck := newTXAckFromTXPKACK(mac, 12345, ack)
				So(txAck, ShouldResemble, gw.TXAck{
					MAC:   mac,
					Token: 12345,
				})
			})
		})
	})
}
