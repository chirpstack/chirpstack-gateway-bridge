package structs

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
)

// UplinkDataFrame implements the uplink data-frame message.
type UplinkDataFrame struct {
	RadioMetaData

	MessageType MessageType `json:"msgtype"`
	MHDR        uint8       `json:"Mhdr"`
	DevAddr     int32       `json:"DevAddr"`
	FCtrl       uint8       `json:"FCtrl"`
	FCnt        uint16      `json:"FCnt"`
	FOpts       string      `json:"FOpts"`
	FPort       int         `json:"FPort"`
	FRMPayload  string      `json:"FRMPayload"`
	MIC         int32       `json:"MIC"`
}

// UplinkDataFrameToProto converts the UplinkDataFrame to the protobuf struct.
func UplinkDataFrameToProto(loraBand band.Band, gatewayID lorawan.EUI64, updf UplinkDataFrame) (gw.UplinkFrame, error) {
	var pb gw.UplinkFrame
	if err := SetRadioMetaDataToProto(loraBand, gatewayID, updf.RadioMetaData, &pb); err != nil {
		return pb, errors.Wrap(err, "set radio meta-data error")
	}

	// MHDR
	pb.PhyPayload = append(pb.PhyPayload, updf.MHDR)

	// devAddr
	devAddr := make([]byte, 4)
	binary.LittleEndian.PutUint32(devAddr, uint32(updf.DevAddr))
	pb.PhyPayload = append(pb.PhyPayload, devAddr...)

	// fCtrl
	pb.PhyPayload = append(pb.PhyPayload, updf.FCtrl)

	// FCnt
	fCnt := make([]byte, 2)
	binary.LittleEndian.PutUint16(fCnt, updf.FCnt)
	pb.PhyPayload = append(pb.PhyPayload, fCnt...)

	// FOpts
	b, err := hex.DecodeString(updf.FOpts)
	if err != nil {
		return pb, errors.Wrap(err, "decode FOpts error")
	}
	pb.PhyPayload = append(pb.PhyPayload, b...)

	// FPort
	if updf.FPort != -1 {
		pb.PhyPayload = append(pb.PhyPayload, uint8(updf.FPort))

		// FRPPayload
		if len(updf.FRMPayload) != 0 {
			b, err = hex.DecodeString(updf.FRMPayload)
			if err != nil {
				return pb, errors.Wrap(err, "decode FRMPayload error")
			}
			pb.PhyPayload = append(pb.PhyPayload, b...)
		}
	}

	// MIC
	mic := make([]byte, 4)
	binary.LittleEndian.PutUint32(mic, uint32(updf.MIC))
	pb.PhyPayload = append(pb.PhyPayload, mic...)

	return pb, nil
}
