package gateway

import (
	"errors"
	"testing"
	"time"

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
