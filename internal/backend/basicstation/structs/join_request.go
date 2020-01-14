package structs

import (
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
)

// JoinRequest implements the join-request message.
type JoinRequest struct {
	RadioMetaData

	MessageType MessageType `json:"msgType"`
	MHDR        uint8       `json:"Mhdr"`
	JoinEUI     EUI64       `json:"JoinEui"`
	DevEUI      EUI64       `json:"DevEui"`
	DevNonce    uint16      `json:"DevNonce"`
	MIC         int32       `json:"MIC"`
}

// JoinRequestToProto converts the JoinRequest to the protobuf struct.
func JoinRequestToProto(loraBand band.Band, gatewayID lorawan.EUI64, jr JoinRequest) (gw.UplinkFrame, error) {
	var pb gw.UplinkFrame
	if err := SetRadioMetaDataToProto(loraBand, gatewayID, jr.RadioMetaData, &pb); err != nil {
		return pb, errors.Wrap(err, "set radio meta-data error")
	}

	// MHDR
	pb.PhyPayload = append(pb.PhyPayload, jr.MHDR)

	// JoinEUI (little endian)
	joinEUI := make([]byte, len(jr.JoinEUI))
	for i := 0; i < len(jr.JoinEUI); i++ {
		joinEUI[len(jr.JoinEUI)-1-i] = jr.JoinEUI[i]
	}
	pb.PhyPayload = append(pb.PhyPayload, joinEUI...)

	// DevEUI (little endian)
	devEUI := make([]byte, len(jr.DevEUI))
	for i := 0; i < len(jr.DevEUI); i++ {
		devEUI[len(jr.DevEUI)-1-i] = jr.DevEUI[i]
	}
	pb.PhyPayload = append(pb.PhyPayload, devEUI...)

	// DevNonce
	devNonce := make([]byte, 2)
	binary.LittleEndian.PutUint16(devNonce, jr.DevNonce)
	pb.PhyPayload = append(pb.PhyPayload, devNonce...)

	// MIC
	mic := make([]byte, 4)
	binary.LittleEndian.PutUint32(mic, uint32(jr.MIC))
	pb.PhyPayload = append(pb.PhyPayload, mic...)

	return pb, nil
}
