package gateway

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDatR(t *testing.T) {
	Convey("Given an empty DatR", t, func() {
		var d DatR

		Convey("Then MarshalJSON returns '0'", func() {
			b, err := d.MarshalJSON()
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, "0")
		})

		Convey("Given LoRa=SF7BW125", func() {
			d.LoRa = "SF7BW125"
			Convey("Then MarshalJSON returns '\"SF7BW125\"'", func() {
				b, err := d.MarshalJSON()
				So(err, ShouldBeNil)
				So(string(b), ShouldEqual, `"SF7BW125"`)
			})
		})

		Convey("Given FSK=1234", func() {
			d.FSK = 1234
			Convey("Then MarshalJSON returns '1234'", func() {
				b, err := d.MarshalJSON()
				So(err, ShouldBeNil)
				So(string(b), ShouldEqual, "1234")
			})
		})

		Convey("Given the string '1234'", func() {
			s := "1234"
			Convey("Then UnmarshalJSON returns FSK=1234", func() {
				err := d.UnmarshalJSON([]byte(s))
				So(err, ShouldBeNil)
				So(d.FSK, ShouldEqual, 1234)
			})
		})

		Convey("Given the string '\"SF7BW125\"'", func() {
			s := `"SF7BW125"`
			Convey("Then UnmarshalJSON returns LoRa=SF7BW125", func() {
				err := d.UnmarshalJSON([]byte(s))
				So(err, ShouldBeNil)
				So(d.LoRa, ShouldEqual, "SF7BW125")
			})
		})
	})
}

func TestCompactTime(t *testing.T) {
	Convey("Given the date 'Mon Jan 2 15:04:05 -0700 MST 2006'", t, func() {
		tStr := "Mon Jan 2 15:04:05 -0700 MST 2006"
		ts, err := time.Parse(tStr, tStr)
		So(err, ShouldBeNil)

		Convey("MarshalJSON returns '\"2006-01-02T22:04:05Z\"'", func() {

			b, err := CompactTime(ts).MarshalJSON()
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, `"2006-01-02T22:04:05Z"`)
		})

		Convey("Given the JSON value of the date (\"2006-01-02T22:04:05Z\")", func() {
			s := `"2006-01-02T22:04:05Z"`
			Convey("UnmarshalJSON returns the correct date", func() {
				var ct CompactTime
				err := ct.UnmarshalJSON([]byte(s))
				So(err, ShouldBeNil)
				So(time.Time(ct).Equal(ts), ShouldBeTrue)
			})
		})
	})

	Convey("Given an empty string as date value", t, func() {
		Convey("UnmarshalJSON returns nil", func() {
			var ct CompactTime
			err := ct.UnmarshalJSON([]byte(`""`))
			So(err, ShouldBeNil)
			So(time.Time(ct).Equal(time.Time{}), ShouldBeTrue)
		})

		Convey("MarshalJSON returns null", func() {
			ct := CompactTime(time.Time{})
			b, err := ct.MarshalJSON()
			So(err, ShouldBeNil)
			So(string(b), ShouldEqual, "null")
		})
	})
}

func TestGetPacketType(t *testing.T) {
	Convey("Given an empty slice []byte{}", t, func() {
		var b []byte

		Convey("Then GetPacketType returns an error (length)", func() {
			_, err := GetPacketType(b)
			So(err, ShouldResemble, errors.New("gateway: at least 4 bytes of data are expected"))
		})

		Convey("Given the slice []byte{3, 1, 3, 4}", func() {
			b = []byte{3, 1, 3, 4}
			Convey("Then GetPacketType returns an error (protocol version)", func() {
				_, err := GetPacketType(b)
				So(err, ShouldResemble, ErrInvalidProtocolVersion)
			})
		})

		Convey("Given the slice []byte{2, 1, 3, 4}", func() {
			b = []byte{2, 1, 3, 4}
			Convey("Then GetPacketType returns PullACK", func() {
				t, err := GetPacketType(b)
				So(err, ShouldBeNil)
				So(t, ShouldEqual, PullACK)
			})
		})
	})
}

func TestPushDataPacket(t *testing.T) {
	Convey("Given an empty PushDataPacket", t, func() {
		var p PushDataPacket
		Convey("Then MarshalBinary returns []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 123, 125}", func() {
			b, err := p.MarshalBinary()
			So(err, ShouldBeNil)
			So(b, ShouldResemble, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 123, 125})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123, GatewayMAC=[]{2, 2, 3, 4, 5, 6, 7, 8}", func() {
			p = PushDataPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			}
			Convey("Then MarshalBinary returns []byte{2, 123, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 123, 125}", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte{2, 123, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 123, 125})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 123, 125}", func() {
			b := []byte{2, 123, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 123, 125}
			Convey("Then UnmarshalBinary returns RandomToken=123, GatewayMAC=[]{1, 2, 3, 4, 5, 6, 7, 8}", func() {
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p, ShouldResemble, PushDataPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     123,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				})
			})
		})
	})
}

func TestPushACKPacket(t *testing.T) {
	Convey("Given an empty PushACKPacket", t, func() {
		var p PushACKPacket
		Convey("Then MarshalBinary returns []byte{0, 0, 0, 1}", func() {
			b, err := p.MarshalBinary()
			So(err, ShouldBeNil)
			So(b, ShouldResemble, []byte{0, 0, 0, 1})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123", func() {
			p = PushACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			}
			Convey("Then MarshalBinary returns []byte{2, 123, 0, 1}", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte{2, 123, 0, 1})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 1}", func() {
			Convey("Then UnmarshalBinary returns RandomToken=123", func() {
				b := []byte{2, 123, 0, 1}
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p, ShouldResemble, PushACKPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     123,
				})
			})
		})
	})
}

func TestPullDataPacket(t *testing.T) {
	Convey("Given an empty PullDataPacket", t, func() {
		var p PullDataPacket
		Convey("Then MarshalBinary returns []byte{0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0}", func() {
			b, err := p.MarshalBinary()
			So(err, ShouldBeNil)
			So(b, ShouldResemble, []byte{0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123, GatewayMAC=[]byte{1, 2, 3, 4, 5, 6, 8, 8}", func() {
			p = PullDataPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
				GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
			}
			Convey("Then MarshalBinary returns []byte{2, 123, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8}", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte{2, 123, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8}", func() {
			b := []byte{2, 123, 0, 2, 1, 2, 3, 4, 5, 6, 7, 8}
			Convey("Then UnmarshalBinary returns RandomToken=123, GatewayMAC=[]byte{1, 2, 3, 4, 5, 6, 8, 8}", func() {
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p, ShouldResemble, PullDataPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     123,
					GatewayMAC:      [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				})
			})
		})
	})
}

func TestPullACKPacket(t *testing.T) {
	Convey("Given an empty PullACKPacket", t, func() {
		var p PullACKPacket
		Convey("Then MarshalBinary returns []byte{0, 0, 0, 4}", func() {
			b, err := p.MarshalBinary()
			So(err, ShouldBeNil)
			So(b, ShouldResemble, []byte{0, 0, 0, 4})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123}", func() {
			p = PullACKPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			}
			Convey("Then MarshalBinary returns []byte{2, 123, 0, 4}", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte{2, 123, 0, 4})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 4}", func() {
			b := []byte{2, 123, 0, 4}
			Convey("Then UnmarshalBinary returns RandomToken=123", func() {
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p, ShouldResemble, PullACKPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     123,
				})
			})
		})
	})
}

func TestPullRespPacket(t *testing.T) {
	Convey("Given an empty PullRespPacket", t, func() {
		var p PullRespPacket
		Convey("Then MarshalBinary returns []byte{0, 0, 0, 3} as first 4 bytes", func() {
			b, err := p.MarshalBinary()
			So(err, ShouldBeNil)
			So(b[0:4], ShouldResemble, []byte{0, 0, 0, 3})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123", func() {
			p = PullRespPacket{
				ProtocolVersion: ProtocolVersion2,
				RandomToken:     123,
			}
			Convey("Then MarshalBinary returns []byte{2, 123, 0, 3} as first 4 bytes", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b[0:4], ShouldResemble, []byte{2, 123, 0, 3})
			})
		})

		Convey("Given ProtocolVersion=1, RandomToken=123", func() {
			p = PullRespPacket{
				ProtocolVersion: ProtocolVersion1,
				RandomToken:     123,
			}
			Convey("Then MarshalBinary returns []byte{1, 0, 0, 3} as first 4 bytes", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b[0:4], ShouldResemble, []byte{1, 0, 0, 3})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 3, 123, 125}", func() {
			b := []byte{2, 123, 0, 3, 123, 125}
			Convey("Then UnmarshalBinary returns RandomToken=123", func() {
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p, ShouldResemble, PullRespPacket{
					ProtocolVersion: ProtocolVersion2,
					RandomToken:     123,
				})
			})
		})
	})
}

func TestTXACKPacket(t *testing.T) {
	Convey("Given an empty TXACKPacket", t, func() {
		var p TXACKPacket
		Convey("Then MarshalBinary returns []byte{0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0}", func() {
			b, err := p.MarshalBinary()
			So(err, ShouldBeNil)
			So(b, ShouldResemble, []byte{0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123 and GatewayMAC=[]byte{8, 7, 6, 5, 4, 3, 2, 1}", func() {
			p.ProtocolVersion = ProtocolVersion2
			p.RandomToken = 123
			p.GatewayMAC = [8]byte{8, 7, 6, 5, 4, 3, 2, 1}
			Convey("Then MarshalBinary returns []byte{2, 123, 0, 5, 8, 7, 6, 5, 4, 3, 2, 1}", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte{2, 123, 0, 5, 8, 7, 6, 5, 4, 3, 2, 1})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 5, 8, 7, 6, 5, 4, 3, 2, 1}", func() {
			b := []byte{2, 123, 0, 5, 8, 7, 6, 5, 4, 3, 2, 1}

			Convey("Then UnmarshalBinary return RandomToken=123 and GatewayMAC=[8]byte{8, 7, 6, 5, 4, 3, 2, 1}", func() {
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p.RandomToken, ShouldEqual, 123)
				So(p.GatewayMAC[:], ShouldResemble, []byte{8, 7, 6, 5, 4, 3, 2, 1})
				So(p.Payload, ShouldBeNil)
				So(p.ProtocolVersion, ShouldEqual, ProtocolVersion2)
			})
		})

		Convey("Given ProtocolVersion=2, RandomToken=123 and a payload with Error=COLLISION_BEACON", func() {
			p.ProtocolVersion = ProtocolVersion2
			p.RandomToken = 123
			p.Payload = &TXACKPayload{
				TXPKACK: TXPKACK{
					Error: "COLLISION_BEACON",
				},
			}

			Convey("Then MarshalBinary returns []byte{2, 123, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 123, 34, 116, 120, 112, 107, 95, 97, 99, 107, 34, 58, 123, 34, 101, 114, 114, 111, 114, 34, 58, 34, 67, 79, 76, 76, 73, 83, 73, 79, 78, 95, 66, 69, 65, 67, 79, 78, 34, 125, 125}", func() {
				b, err := p.MarshalBinary()
				So(err, ShouldBeNil)
				So(b, ShouldResemble, []byte{2, 123, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 123, 34, 116, 120, 112, 107, 95, 97, 99, 107, 34, 58, 123, 34, 101, 114, 114, 111, 114, 34, 58, 34, 67, 79, 76, 76, 73, 83, 73, 79, 78, 95, 66, 69, 65, 67, 79, 78, 34, 125, 125})
			})
		})

		Convey("Given the slice []byte{2, 123, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 123, 34, 116, 120, 112, 107, 95, 97, 99, 107, 34, 58, 123, 34, 101, 114, 114, 111, 114, 34, 58, 34, 67, 79, 76, 76, 73, 83, 73, 79, 78, 95, 66, 69, 65, 67, 79, 78, 34, 125, 125}", func() {
			b := []byte{2, 123, 0, 5, 0, 0, 0, 0, 0, 0, 0, 0, 123, 34, 116, 120, 112, 107, 95, 97, 99, 107, 34, 58, 123, 34, 101, 114, 114, 111, 114, 34, 58, 34, 67, 79, 76, 76, 73, 83, 73, 79, 78, 95, 66, 69, 65, 67, 79, 78, 34, 125, 125}
			Convey("Then UnmarshalBinary returns RandomToken=123 and a payload with Error=COLLISION_BEACON", func() {
				err := p.UnmarshalBinary(b)
				So(err, ShouldBeNil)
				So(p.RandomToken, ShouldEqual, 123)
				So(p.ProtocolVersion, ShouldEqual, ProtocolVersion2)
				So(p.Payload, ShouldResemble, &TXACKPayload{
					TXPKACK: TXPKACK{
						Error: "COLLISION_BEACON",
					},
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
