package semtechudp

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/stats"
	"github.com/brocaar/lorawan"
)

// errors
var (
	errGatewayDoesNotExist = errors.New("gateway does not exist")
)

// gatewayCleanupDuration contains the duration after which the gateway is
// cleaned up from the registry after no activity
var gatewayCleanupDuration = -1 * time.Minute

// gateway contains a connection and meta-data for a gateway connection.
type gateway struct {
	stats           *stats.Collector
	addr            *net.UDPAddr
	lastSeen        time.Time
	protocolVersion uint8
}

// gateways contains the gateways registry.
type gateways struct {
	sync.RWMutex
	gateways map[lorawan.EUI64]gateway

	subscribeEventFunc func(events.Subscribe)
}

// get returns the gateway object for the given MAC.
func (c *gateways) get(mac lorawan.EUI64) (gateway, error) {
	c.RLock()
	defer c.RUnlock()

	gw, ok := c.gateways[mac]
	if !ok {
		return gw, errGatewayDoesNotExist
	}

	return gw, nil
}

// Set creates or updates the gateway for the given Gateway ID.
// Note that set must only be called for PullData frames! The UDP Packet
// Forwarded uses two UDP sockets and the socket responsible for sending the
// PullData is used for receiving downlink data.
func (c *gateways) set(gatewayID lorawan.EUI64, gw gateway) error {
	c.Lock()
	defer c.Unlock()

	gww, ok := c.gateways[gatewayID]
	if !ok {
		gw.stats = stats.NewCollector()
		connectCounter().Inc()
	} else {
		gw.stats = gww.stats
	}

	if c.subscribeEventFunc != nil {
		c.subscribeEventFunc(events.Subscribe{
			Subscribe: true,
			GatewayID: gatewayID,
		})
	}

	c.gateways[gatewayID] = gw
	return nil
}

// cleanup removes inactive gateways from the registry.
func (c *gateways) cleanup() error {
	c.Lock()
	defer c.Unlock()

	for gatewayID := range c.gateways {
		if c.gateways[gatewayID].lastSeen.Before(time.Now().Add(gatewayCleanupDuration)) {
			disconnectCounter().Inc()

			if c.subscribeEventFunc != nil {
				c.subscribeEventFunc(events.Subscribe{
					Subscribe: false,
					GatewayID: gatewayID,
				})
			}

			delete(c.gateways, gatewayID)
		}
	}
	return nil
}
