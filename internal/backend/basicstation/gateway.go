package basicstation

import (
	"errors"
	"sync"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/lorawan"
	"github.com/gorilla/websocket"
)

var (
	errGatewayDoesNotExist = errors.New("gateway does not exist")
)

type gateway struct {
	conn          *websocket.Conn
	configVersion string
}

type gateways struct {
	sync.RWMutex
	gateways map[lorawan.EUI64]gateway

	subscribeEventChan chan events.Subscribe
}

func (g *gateways) get(id lorawan.EUI64) (gateway, error) {
	g.RLock()
	defer g.RUnlock()

	gw, ok := g.gateways[id]
	if !ok {
		return gw, errGatewayDoesNotExist
	}
	return gw, nil
}

func (g *gateways) set(id lorawan.EUI64, gw gateway) error {
	g.Lock()
	defer g.Unlock()

	g.gateways[id] = gw
	g.subscribeEventChan <- events.Subscribe{Subscribe: true, GatewayID: id}
	return nil
}

func (g *gateways) remove(id lorawan.EUI64) error {
	g.Lock()
	defer g.Unlock()

	g.subscribeEventChan <- events.Subscribe{Subscribe: false, GatewayID: id}
	delete(g.gateways, id)
	return nil
}
