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
		backend, err := NewBackend("127.0.0.1:0", nil, nil)
		So(err, ShouldBeNil)

		backendAddr, err := net.ResolveUDPAddr("udp", backend.conn.LocalAddr().String())
		So(err, ShouldBeNil)

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

			Convey("When sending a PUSH_DATA packet with stats", func() {
				p := PushDataPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     1234,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Payload: PushDataPayload{
						Stat: &Stat{
							Time: ExpandedTime(time.Time{}.UTC()),
							Lati: 1.234,
							Long: 2.123,
							Alti: 123,
							RXNb: 1,
							RXOK: 2,
							RXFW: 3,
							ACKR: 33.3,
							DWNb: 4,
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

			Convey("When sending a PUSH_DATA packet with RXPK", func() {
				p := PushDataPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     1234,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
					Payload: PushDataPayload{
						RXPK: []RXPK{
							{
								Time: CompactTime(time.Now().UTC()),
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

					rxPacket2, err := newRXPacketFromRXPK(p.GatewayMAC, p.Payload.RXPK[0])
					So(err, ShouldBeNil)
					So(rxPacket, ShouldResemble, rxPacket2)
				})
			})

			Convey("Given a TXPacket", func() {
				txPacket := gw.TXPacketBytes{
					TXInfo: gw.TXInfo{
						MAC:         [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
						Immediately: true,
						Timestamp:   12345,
						Frequency:   868100000,
						Power:       14,
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

						So(pullResp, ShouldResemble, PullRespPacket{
							ProtocolVersion: p.ProtocolVersion,
							Payload: PullRespPayload{
								TXPK: TXPK{
									Imme: true,
									Tmst: 12345,
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
	Convey("Given a (Semtech) Stat struct and gateway MAC", t, func() {
		now := time.Now().UTC()
		stat := Stat{
			Time: ExpandedTime(now),
			Lati: 1.234,
			Long: 2.123,
			Alti: 234,
			RXNb: 1,
			RXOK: 2,
			RXFW: 3,
			ACKR: 33.3,
			DWNb: 4,
		}
		mac := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}

		Convey("When calling newGatewayStatsPacket", func() {
			gwStats := newGatewayStatsPacket(mac, stat)
			Convey("Then all fields are set correctly", func() {
				So(gwStats, ShouldResemble, gw.GatewayStatsPacket{
					Time:                now,
					MAC:                 mac,
					Latitude:            1.234,
					Longitude:           2.123,
					Altitude:            234,
					RXPacketsReceived:   1,
					RXPacketsReceivedOK: 2,
				})
			})
		})

	})
}

func TestNewTXPKFromTXPacket(t *testing.T) {
	Convey("Given a TXPacket", t, func() {
		txPacket := gw.TXPacketBytes{
			TXInfo: gw.TXInfo{
				Timestamp: 12345,
				Frequency: 868100000,
				Power:     14,
				CodeRate:  "4/5",
				DataRate: band.DataRate{
					Modulation:   band.LoRaModulation,
					SpreadFactor: 9,
					Bandwidth:    250,
				},
			},
			PHYPayload: []byte{1, 2, 3, 4},
		}

		Convey("Then te expected TXPK is returned (with default IPol", func() {
			txpk, err := newTXPKFromTXPacket(txPacket)
			So(err, ShouldBeNil)
			So(txpk, ShouldResemble, TXPK{
				Imme: false,
				Tmst: 12345,
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
	Convey("Given a (Semtech) RXPK and gateway MAC", t, func() {
		now := time.Now().UTC()
		rxpk := RXPK{
			Time: CompactTime(now),
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

		Convey("When calling newRXPacketFromRXPK(", func() {
			rxPacket, err := newRXPacketFromRXPK(mac, rxpk)
			So(err, ShouldBeNil)

			Convey("Then all fields are set correctly", func() {
				So(rxPacket.PHYPayload, ShouldResemble, []byte{1, 2, 3, 4})

				So(rxPacket.RXInfo, ShouldResemble, gw.RXInfo{
					MAC:       mac,
					Time:      now,
					Timestamp: 708016819,
					Frequency: 868500000,
					Channel:   2,
					RFChain:   1,
					CRCStatus: 1,
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
