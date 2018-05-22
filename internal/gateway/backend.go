package gateway

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
)

// errors
var (
	errGatewayDoesNotExist = errors.New("gateway does not exist")
)

// gatewayCleanupDuration contains the duration after which the gateway is
// cleaned up from the registry after no activity
var gatewayCleanupDuration = -1 * time.Minute

// loRaDataRateRegex contains a regexp for parsing the data-rate string.
var loRaDataRateRegex = regexp.MustCompile(`SF(\d+)BW(\d+)`)

// udpPacket represents a raw UDP packet.
type udpPacket struct {
	addr *net.UDPAddr
	data []byte
}

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
	defer c.RUnlock()
	c.RLock()
	gw, ok := c.gateways[mac]
	if !ok {
		return gw, errGatewayDoesNotExist
	}
	return gw, nil
}

// set creates or updates the gateway for the given MAC.
func (c *gateways) set(mac lorawan.EUI64, gw gateway) error {
	defer c.Unlock()
	c.Lock()
	_, ok := c.gateways[mac]
	if !ok && c.onNew != nil {
		if err := c.onNew(mac); err != nil {
			return err
		}
	}
	c.gateways[mac] = gw
	return nil
}

// cleanup removes inactive gateways from the registry.
func (c *gateways) cleanup() error {
	defer c.Unlock()
	c.Lock()
	for mac := range c.gateways {
		if c.gateways[mac].lastSeen.Before(time.Now().Add(gatewayCleanupDuration)) {
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

// Configuration holds the packet-forwarder configuration.
type Configuration struct {
	MAC            lorawan.EUI64 `mapstructure:"-"`
	MACString      string        `mapstructure:"mac"`
	BaseFile       string        `mapstructure:"base_file"`
	OutputFile     string        `mapstructure:"output_file"`
	RestartCommand string        `mapstructure:"restart_command"`
	version        string
}

// Backend implements a Semtech packet-forwarder gateway backend.
type Backend struct {
	conn           *net.UDPConn
	txAckChan      chan gw.TXAck              // received TX ACKs
	rxChan         chan gw.RXPacketBytes      // received uplink frames
	statsChan      chan gw.GatewayStatsPacket // received gateway stats
	udpSendChan    chan udpPacket
	closed         bool
	gateways       gateways
	configurations []Configuration
	wg             sync.WaitGroup
	skipCRCCheck   bool
}

// NewBackend creates a new backend.
func NewBackend(bind string, onNew func(lorawan.EUI64) error, onDelete func(lorawan.EUI64) error, skipCRCCheck bool, configurations []Configuration) (*Backend, error) {
	addr, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		return nil, err
	}
	log.WithField("addr", addr).Info("gateway: starting gateway udp listener")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	b := &Backend{
		skipCRCCheck: skipCRCCheck,
		conn:         conn,
		txAckChan:    make(chan gw.TXAck),
		rxChan:       make(chan gw.RXPacketBytes),
		statsChan:    make(chan gw.GatewayStatsPacket),
		udpSendChan:  make(chan udpPacket),
		gateways: gateways{
			gateways: make(map[lorawan.EUI64]gateway),
			onNew:    onNew,
			onDelete: onDelete,
		},
		configurations: configurations,
	}

	go func() {
		for {
			if err := b.gateways.cleanup(); err != nil {
				log.Errorf("gateway: gateways cleanup failed: %s", err)
			}
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		b.wg.Add(1)
		err := b.readPackets()
		if !b.closed {
			log.Error(err)
		}
		b.wg.Done()
	}()

	go func() {
		b.wg.Add(1)
		err := b.sendPackets()
		if !b.closed {
			log.Error(err)
		}
		b.wg.Done()
	}()

	return b, nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	log.Info("gateway: closing gateway backend")
	b.closed = true
	close(b.udpSendChan)
	if err := b.conn.Close(); err != nil {
		return err
	}
	log.Info("gateway: handling last packets")
	b.wg.Wait()
	return nil
}

// ApplyConfiguration applies the given configuration.
func (b *Backend) ApplyConfiguration(config gw.GatewayConfigPacket) error {
	var found bool
	for i, c := range b.configurations {
		if c.MAC != config.MAC {
			continue
		}

		found = true

		gwConfig, err := getGatewayConfig(config)
		if err != nil {
			return errors.Wrap(err, "get gateway config error")
		}

		baseConfig, err := loadConfigFile(c.BaseFile)
		if err != nil {
			return errors.Wrap(err, "load config file error")
		}

		if err = mergeConfig(c.MAC, baseConfig, gwConfig); err != nil {
			return errors.Wrap(err, "merge config error")
		}

		// generate config json
		bb, err := json.Marshal(baseConfig)
		if err != nil {
			return errors.Wrap(err, "marshal json error")
		}

		// write new config file to disk
		if err = ioutil.WriteFile(c.OutputFile, bb, 0644); err != nil {
			return errors.Wrap(err, "write config file errror")
		}
		log.WithFields(log.Fields{
			"mac":  config.MAC,
			"file": c.OutputFile,
		}).Info("gateway: new configuration file written")

		// invoke restart command
		if err = invokePFRestart(c.RestartCommand); err != nil {
			return errors.Wrap(err, "invoke packet-forwarder restart error")
		}
		log.WithFields(log.Fields{
			"mac": config.MAC,
			"cmd": c.RestartCommand,
		}).Info("gateway: packet-forwarder restart command invoked")

		b.configurations[i].version = config.Version
	}

	if !found {
		log.WithField("mac", config.MAC).Warning("gateway: configuration was not applied, gateway is not configured for managed configuration")
	}

	return nil
}

// RXPacketChan returns the channel containing the received RX packets.
func (b *Backend) RXPacketChan() chan gw.RXPacketBytes {
	return b.rxChan
}

// StatsChan returns the channel containing the received gateway stats.
func (b *Backend) StatsChan() chan gw.GatewayStatsPacket {
	return b.statsChan
}

// TXAckChan returns the channel containing the TX acknowledgements
// (or errors).
func (b *Backend) TXAckChan() chan gw.TXAck {
	return b.txAckChan
}

// Send sends the given packet to the gateway.
func (b *Backend) Send(txPacket gw.TXPacketBytes) error {
	gw, err := b.gateways.get(txPacket.TXInfo.MAC)
	if err != nil {
		return err
	}
	txpk, err := newTXPKFromTXPacket(txPacket)
	if err != nil {
		return err
	}
	pullResp := PullRespPacket{
		RandomToken:     txPacket.Token,
		ProtocolVersion: gw.protocolVersion,
		Payload: PullRespPayload{
			TXPK: txpk,
		},
	}
	bytes, err := pullResp.MarshalBinary()
	if err != nil {
		return fmt.Errorf("gateway: json marshall PullRespPacket error: %s", err)
	}
	b.udpSendChan <- udpPacket{
		data: bytes,
		addr: gw.addr,
	}
	return nil
}

func (b *Backend) readPackets() error {
	buf := make([]byte, 65507) // max udp data size
	for {
		i, addr, err := b.conn.ReadFromUDP(buf)
		if err != nil {
			if b.closed {
				return nil
			}

			log.WithError(err).Error("gateway: read from udp error")
			continue
		}
		data := make([]byte, i)
		copy(data, buf[:i])
		go func(data []byte) {
			if err := b.handlePacket(addr, data); err != nil {
				log.WithFields(log.Fields{
					"data_base64": base64.StdEncoding.EncodeToString(data),
					"addr":        addr,
				}).Errorf("gateway: could not handle packet: %s", err)
			}
		}(data)
	}
}

func (b *Backend) sendPackets() error {
	for p := range b.udpSendChan {
		pt, err := GetPacketType(p.data)
		if err != nil {
			log.WithFields(log.Fields{
				"addr":        p.addr,
				"data_base64": base64.StdEncoding.EncodeToString(p.data),
			}).Error("gateway: unknown packet type")
			continue
		}
		log.WithFields(log.Fields{
			"addr":             p.addr,
			"type":             pt,
			"protocol_version": p.data[0],
		}).Info("gateway: sending udp packet to gateway")

		if _, err := b.conn.WriteToUDP(p.data, p.addr); err != nil {
			log.WithFields(log.Fields{
				"addr":             p.addr,
				"type":             pt,
				"protocol_version": p.data[0],
			}).WithError(err).Error("gateway: write to udp error")
		}
	}
	return nil
}

func (b *Backend) handlePacket(addr *net.UDPAddr, data []byte) error {
	pt, err := GetPacketType(data)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"addr":             addr,
		"type":             pt,
		"protocol_version": data[0],
	}).Info("gateway: received udp packet from gateway")

	switch pt {
	case PushData:
		return b.handlePushData(addr, data)
	case PullData:
		return b.handlePullData(addr, data)
	case TXACK:
		return b.handleTXACK(addr, data)
	default:
		return fmt.Errorf("gateway: unknown packet type: %s", pt)
	}
}

func (b *Backend) handlePullData(addr *net.UDPAddr, data []byte) error {
	var p PullDataPacket
	if err := p.UnmarshalBinary(data); err != nil {
		return err
	}
	ack := PullACKPacket{
		ProtocolVersion: p.ProtocolVersion,
		RandomToken:     p.RandomToken,
	}
	bytes, err := ack.MarshalBinary()
	if err != nil {
		return err
	}

	err = b.gateways.set(p.GatewayMAC, gateway{
		addr:            addr,
		lastSeen:        time.Now().UTC(),
		protocolVersion: p.ProtocolVersion,
	})
	if err != nil {
		return err
	}

	b.udpSendChan <- udpPacket{
		addr: addr,
		data: bytes,
	}
	return nil
}

func (b *Backend) handlePushData(addr *net.UDPAddr, data []byte) error {
	var p PushDataPacket
	if err := p.UnmarshalBinary(data); err != nil {
		return err
	}

	// ack the packet
	ack := PushACKPacket{
		ProtocolVersion: p.ProtocolVersion,
		RandomToken:     p.RandomToken,
	}
	bytes, err := ack.MarshalBinary()
	if err != nil {
		return err
	}
	b.udpSendChan <- udpPacket{
		addr: addr,
		data: bytes,
	}

	// gateway stats
	if p.Payload.Stat != nil {
		b.handleStat(addr, p.GatewayMAC, *p.Payload.Stat)
	}

	// rx packets
	for _, rxpk := range p.Payload.RXPK {
		if err := b.handleRXPacket(addr, p.GatewayMAC, rxpk); err != nil {
			return err
		}
	}
	return nil
}

func (b *Backend) handleStat(addr *net.UDPAddr, mac lorawan.EUI64, stat Stat) {
	gwStats := newGatewayStatsPacket(mac, stat)
	log.WithFields(log.Fields{
		"addr": addr,
		"mac":  mac,
	}).Info("gateway: stat packet received")

	// set configuration version, if available
	for _, c := range b.configurations {
		if gwStats.MAC == c.MAC {
			gwStats.ConfigVersion = c.version
		}
	}

	b.statsChan <- gwStats
}

func (b *Backend) handleRXPacket(addr *net.UDPAddr, mac lorawan.EUI64, rxpk RXPK) error {
	logFields := log.Fields{
		"addr": addr,
		"mac":  mac,
		"data": rxpk.Data,
	}
	log.WithFields(logFields).Info("gateway: rxpk packet received")

	// decode packet(s)
	rxPackets, err := newRXPacketsFromRXPK(mac, rxpk)
	if err != nil {
		return err
	}

	for i := range rxPackets {
		// check CRC
		if !b.skipCRCCheck && rxPackets[i].RXInfo.CRCStatus != 1 {
			log.WithFields(logFields).Warningf("gateway: invalid packet CRC: %d", rxPackets[i].RXInfo.CRCStatus)
			return errors.New("gateway: invalid CRC")
		}
		b.rxChan <- rxPackets[i]
	}

	return nil
}

func (b *Backend) handleTXACK(addr *net.UDPAddr, data []byte) error {
	var p TXACKPacket
	if err := p.UnmarshalBinary(data); err != nil {
		return err
	}

	if p.Payload != nil {
		txAck := newTXAckFromTXPKACK(p.GatewayMAC, p.RandomToken, p.Payload.TXPKACK)
		b.txAckChan <- txAck
	} else {
		b.txAckChan <- gw.TXAck{
			MAC:   p.GatewayMAC,
			Token: p.RandomToken,
		}
	}

	return nil
}
