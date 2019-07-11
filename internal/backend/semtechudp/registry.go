package semtechudp

import (
	"errors"
	"net"
	"sync"
	"time"

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
	addr            *net.UDPAddr
	lastSeen        time.Time
	protocolVersion uint8
}

// gateways contains the gateways registry.
type gateways struct {
	sync.RWMutex
	gateways map[lorawan.EUI64]gateway

	connectChan    chan lorawan.EUI64
	disconnectChan chan lorawan.EUI64
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

// set creates or updates the gateway for the given Gateway ID.
func (c *gateways) set(gatewayID lorawan.EUI64, gw gateway) error {
	c.Lock()
	defer c.Unlock()

	_, ok := c.gateways[gatewayID]
	if !ok {
		connectCounter().Inc()
		c.connectChan <- gatewayID
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
			c.disconnectChan <- gatewayID
			delete(c.gateways, gatewayID)
		}
	}
	return nil
}
