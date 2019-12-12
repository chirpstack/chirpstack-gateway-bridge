package structs

import (
	"encoding/hex"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/pkg/errors"
)

// UplinkProprietaryFrame implements the uplink proprietary frame.
type UplinkProprietaryFrame struct {
	RadioMetaData

	MessageType MessageType `json:"msgType"`
	FRMPayload  string      `json:"FRMPayload"`
}

// UplinkProprietaryFrameToProto converts the UplinkProprietaryFrame to the protobuf struct.
func UplinkProprietaryFrameToProto(loraBand band.Band, gatewayID lorawan.EUI64, uppf UplinkProprietaryFrame) (gw.UplinkFrame, error) {
	var pb gw.UplinkFrame
	if err := SetRadioMetaDataToProto(loraBand, gatewayID, uppf.RadioMetaData, &pb); err != nil {
		return pb, errors.Wrap(err, "set radio meta-data error")
	}

	// FRMPayload is actually the full PHYPayload:
	//
	frmPayload, err := hex.DecodeString(uppf.FRMPayload)
	if err != nil {
		return pb, errors.Wrap(err, "decode FRMPayload field error")
	}
	pb.PhyPayload = append(pb.PhyPayload, frmPayload...)

	return pb, nil
}
