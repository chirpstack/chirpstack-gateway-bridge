package events

import "github.com/brocaar/lorawan"

// Subscribe event
type Subscribe struct {
	// Gateway ID.
	GatewayID lorawan.EUI64

	// Subscribe (true) or unsubscribe (false) the gateway.
	Subscribe bool
}
