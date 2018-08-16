package semtech

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
	onNew    func(lorawan.EUI64) error
	onDelete func(lorawan.EUI64) error
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

// set creates or updates the gateway for the given MAC.
func (c *gateways) set(mac lorawan.EUI64, gw gateway) error {
	c.Lock()
	defer c.Unlock()

	_, ok := c.gateways[mac]
	if !ok && c.onNew != nil {
		gatewayEventCounter("register_gateway")
		if err := c.onNew(mac); err != nil {
			return err
		}
	}
	c.gateways[mac] = gw
	return nil
}

// cleanup removes inactive gateways from the registry.
func (c *gateways) cleanup() error {
	c.Lock()
	defer c.Unlock()

	for mac := range c.gateways {
		if c.gateways[mac].lastSeen.Before(time.Now().Add(gatewayCleanupDuration)) {
			gatewayEventCounter("unregister_gateway")
			if c.onDelete != nil {
				if err := c.onDelete(mac); err != nil {
					return err
				}
			}
			delete(c.gateways, mac)
		}
	}
	return nil
}
