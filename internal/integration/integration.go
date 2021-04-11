package integration

import (
	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/integration/mqtt"
	"github.com/brocaar/lorawan"
)

// Event types.
const (
	EventUp    = "up"
	EventStats = "stats"
	EventAck   = "ack"
	EventRaw   = "raw"
)

var integration Integration

// Setup configures the integration.
func Setup(conf config.Config) error {
	var err error
	integration, err = mqtt.NewBackend(conf)
	if err != nil {
		return errors.Wrap(err, "setup mqtt integration error")
	}

	return nil
}

// GetIntegration returns the integration.
func GetIntegration() Integration {
	return integration
}

// Integration defines the interface that an integration must implement.
type Integration interface {
	// SetGatewaySubscription updates the gateway subscription for the given
	// gateway ID. The integration must implement this such that it is safe
	// to call the same action multiple times.
	SetGatewaySubscription(subscribe bool, gatewayID lorawan.EUI64) error

	// PublishEvent publishes the given event.
	PublishEvent(lorawan.EUI64, string, uuid.UUID, proto.Message) error

	// PublishState publishes the given state as retained message.
	PublishState(lorawan.EUI64, string, proto.Message) error

	// SetDownlinkFrameFunc sets the DownlinkFrame handler func.
	SetDownlinkFrameFunc(func(gw.DownlinkFrame))

	// SetRawPacketForwarderCommandFunc sets the RawPacketForwarderCommand handler func.
	SetRawPacketForwarderCommandFunc(func(gw.RawPacketForwarderCommand))

	// SetGatewayConfigurationFunc sets the GatewayConfiguration handler func.
	SetGatewayConfigurationFunc(func(gw.GatewayConfiguration))

	// SetGatewayCommandExecRequestFunc sets the GatewayCommandExecRequest handler func.
	SetGatewayCommandExecRequestFunc(func(gw.GatewayCommandExecRequest))

	// Start starts the integration.
	Start() error

	// Stop stops the integration.
	Stop() error
}
