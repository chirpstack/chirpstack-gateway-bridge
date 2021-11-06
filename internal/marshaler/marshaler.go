package marshaler

import (
	"bytes"
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)


type Marshaler struct {
	Marshal   func(proto.Message) ([]byte, error)
	Unmarshal func([]byte, proto.Message) error
}


func GetMarshaler(marshaler string) (*Marshaler, error) {
	output := Marshaler {}
	switch marshaler {
	case "json":
		output.Marshal = func(msg proto.Message) ([]byte, error) {
			marshaler := &jsonpb.Marshaler{
				EnumsAsInts:  false,
				EmitDefaults: true,
			}
			str, err := marshaler.MarshalToString(msg)
			return []byte(str), err
		}

		output.Unmarshal = func(b []byte, msg proto.Message) error {
			unmarshaler := &jsonpb.Unmarshaler{
				AllowUnknownFields: true, // we don't want to fail on unknown fields
			}
			return unmarshaler.Unmarshal(bytes.NewReader(b), msg)
		}

		return &output, nil
	case "protobuf":
		output.Marshal = func(msg proto.Message) ([]byte, error) {
			return proto.Marshal(msg)
		}

		output.Unmarshal = func(b []byte, msg proto.Message) error {
			return proto.Unmarshal(b, msg)
		}

		return &output, nil
	default:
		return nil, fmt.Errorf("unknown marshaler: %s", marshaler)
	}
}
