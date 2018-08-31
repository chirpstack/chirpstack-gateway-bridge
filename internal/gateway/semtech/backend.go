package semtech

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/lora-gateway-bridge/internal/gateway/semtech/packets"
	"github.com/brocaar/loraserver/api/gw"
	"github.com/brocaar/lorawan"
)

// udpPacket represents a raw UDP packet.
type udpPacket struct {
	addr *net.UDPAddr
	data []byte
}

// PFConfiguration holds the packet-forwarder configuration.
type PFConfiguration struct {
	MAC            lorawan.EUI64 `mapstructure:"-"`
	MACString      string        `mapstructure:"mac"`
	BaseFile       string        `mapstructure:"base_file"`
	OutputFile     string        `mapstructure:"output_file"`
	RestartCommand string        `mapstructure:"restart_command"`
	Version        string        `mapstructure:"-"`
}

// Backend implements a Semtech packet-forwarder gateway backend.
type Backend struct {
	sync.RWMutex

	downlinkTXAckChan chan gw.DownlinkTXAck
	uplinkFrameChan   chan gw.UplinkFrame
	gatewayStatsChan  chan gw.GatewayStats
	udpSendChan       chan udpPacket

	wg             sync.WaitGroup
	conn           *net.UDPConn
	closed         bool
	gateways       gateways
	configurations []PFConfiguration
}

// NewBackend creates a new backend.
func NewBackend(bind string, onNew, onDelete func(lorawan.EUI64) error, configurations []PFConfiguration) (*Backend, error) {
	addr, err := net.ResolveUDPAddr("udp", bind)
	if err != nil {
		return nil, errors.Wrap(err, "resolve udp addr error")
	}

	log.WithField("addr", addr).Info("gateway: starting gateway udp listener")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, errors.Wrap(err, "listen udp error")
	}

	b := &Backend{
		conn:              conn,
		downlinkTXAckChan: make(chan gw.DownlinkTXAck),
		uplinkFrameChan:   make(chan gw.UplinkFrame),
		gatewayStatsChan:  make(chan gw.GatewayStats),
		udpSendChan:       make(chan udpPacket),
		gateways: gateways{
			gateways: make(map[lorawan.EUI64]gateway),
			onNew:    onNew,
			onDelete: onDelete,
		},
		configurations: configurations,
	}

	go func() {
		for {
			log.Debug("gateway: cleanup gateway registry")
			if err := b.gateways.cleanup(); err != nil {
				log.WithError(err).Error("gateway: gateway registry cleanup failed")
			}
			time.Sleep(time.Minute)
		}
	}()

	go func() {
		b.wg.Add(1)
		err := b.readPackets()
		if !b.isClosed() {
			log.WithError(err).Error("gateway: read udp packets error")
		}
		b.wg.Done()
	}()

	go func() {
		b.wg.Add(1)
		err := b.sendPackets()
		if !b.isClosed() {
			log.WithError(err).Error("gateway: send udp packets error")
		}
		b.wg.Done()
	}()

	return b, nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.Lock()
	b.closed = true

	log.Info("gateway: closing gateway backend")

	if err := b.conn.Close(); err != nil {
		return errors.Wrap(err, "close udp listener error")
	}

	log.Info("gateway: handling last packets")
	close(b.udpSendChan)
	b.Unlock()
	b.wg.Wait()
	return nil
}

// DownlinkTXAckChan returns the downlink tx ack channel.
func (b *Backend) DownlinkTXAckChan() chan gw.DownlinkTXAck {
	return b.downlinkTXAckChan
}

// GatewayStatsChan returns the gateway stats channel.
func (b *Backend) GatewayStatsChan() chan gw.GatewayStats {
	return b.gatewayStatsChan
}

// UplinkFrameChan returns the uplink frame channel.
func (b *Backend) UplinkFrameChan() chan gw.UplinkFrame {
	return b.uplinkFrameChan
}

// SendDownlinkFrame sends the given downlink frame to the gateway.
func (b *Backend) SendDownlinkFrame(frame gw.DownlinkFrame) error {
	var gatewayID lorawan.EUI64
	copy(gatewayID[:], frame.TxInfo.GatewayId)

	gw, err := b.gateways.get(gatewayID)
	if err != nil {
		return errors.Wrap(err, "get gateway error")
	}

	pullResp, err := packets.GetPullRespPacket(gw.protocolVersion, uint16(frame.Token), frame)
	if err != nil {
		return errors.Wrap(err, "get PullRespPacket error")
	}

	bytes, err := pullResp.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "gateway: marshal PullRespPacket error")
	}

	b.udpSendChan <- udpPacket{
		data: bytes,
		addr: gw.addr,
	}
	return nil
}

// ApplyConfiguration applies the given configuration to the gateway
// (packet-forwarder).
func (b *Backend) ApplyConfiguration(config gw.GatewayConfiguration) error {
	return gatewayConfigHandleTimer(func() error {
		var gatewayID lorawan.EUI64
		copy(gatewayID[:], config.GatewayId)

		b.Lock()
		var pfConfig *PFConfiguration
		for i := range b.configurations {
			if b.configurations[i].MAC == gatewayID {
				pfConfig = &b.configurations[i]
			}
		}
		b.Unlock()

		if pfConfig == nil {
			return errGatewayDoesNotExist
		}

		return b.applyConfiguration(*pfConfig, config)
	})
}

func (b *Backend) applyConfiguration(pfConfig PFConfiguration, config gw.GatewayConfiguration) error {
	gwConfig, err := getGatewayConfig(config)
	if err != nil {
		return errors.Wrap(err, "get gateway config error")
	}

	baseConfig, err := loadConfigFile(pfConfig.BaseFile)
	if err != nil {
		return errors.Wrap(err, "load config file error")
	}

	if err = mergeConfig(pfConfig.MAC, baseConfig, gwConfig); err != nil {
		return errors.Wrap(err, "merge config error")
	}

	// generate config json
	bb, err := json.Marshal(baseConfig)
	if err != nil {
		return errors.Wrap(err, "marshal json error")
	}

	// write new config file to disk
	if err = ioutil.WriteFile(pfConfig.OutputFile, bb, 0644); err != nil {
		return errors.Wrap(err, "write config file errror")
	}
	log.WithFields(log.Fields{
		"gateway_id": pfConfig.MAC,
		"file":       pfConfig.OutputFile,
	}).Info("gateway: new configuration file written")

	// invoke restart command
	if err = invokePFRestart(pfConfig.RestartCommand); err != nil {
		return errors.Wrap(err, "invoke packet-forwarder restart error")
	}
	log.WithFields(log.Fields{
		"gateway_id": pfConfig.MAC,
		"cmd":        pfConfig.RestartCommand,
	}).Info("gateway: packet-forwarder restart command invoked")

	b.Lock()
	defer b.Unlock()

	for i := range b.configurations {
		if b.configurations[i].MAC == pfConfig.MAC {
			b.configurations[i].Version = config.Version
		}
	}

	return nil
}

func (b *Backend) isClosed() bool {
	b.RLock()
	defer b.RUnlock()
	return b.closed
}

func (b *Backend) readPackets() error {
	buf := make([]byte, 65507) // max udp data size
	for {
		i, addr, err := b.conn.ReadFromUDP(buf)
		if err != nil {
			if b.isClosed() {
				return nil
			}

			log.WithError(err).Error("gateway: read from udp error")
			continue
		}
		data := make([]byte, i)
		copy(data, buf[:i])
		up := udpPacket{data: data, addr: addr}

		// handle packet async
		go func(up udpPacket) {
			if err := b.handlePacket(up); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"data_base64": base64.StdEncoding.EncodeToString(up.data),
					"addr":        up.addr,
				}).Error("gateway: could not handle packet")
			}
		}(up)
	}
}

func (b *Backend) sendPackets() error {
	for p := range b.udpSendChan {
		pt, err := packets.GetPacketType(p.data)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"addr":        p.addr,
				"data_base64": base64.StdEncoding.EncodeToString(p.data),
			}).Error("gateway: get packet-type error")
			continue
		}

		log.WithFields(log.Fields{
			"addr":             p.addr,
			"type":             pt,
			"protocol_version": p.data[0],
		}).Debug("gateway: sending udp packet to gateway")

		err = gatewayWriteUDPTimer(pt.String(), func() error {
			_, err := b.conn.WriteToUDP(p.data, p.addr)
			return err
		})

		if err != nil {
			log.WithFields(log.Fields{
				"addr":             p.addr,
				"type":             pt,
				"protocol_version": p.data[0],
			}).WithError(err).Error("gateway: write to udp error")
		}
	}
	return nil
}

func (b *Backend) handlePacket(up udpPacket) error {
	b.RLock()
	defer b.RUnlock()

	if b.closed {
		return nil
	}

	pt, err := packets.GetPacketType(up.data)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"addr":             up.addr,
		"type":             pt,
		"protocol_version": up.data[0],
	}).Debug("gateway: received udp packet from gateway")

	return gatewayHandleTimer(pt.String(), func() error {
		switch pt {
		case packets.PushData:
			return b.handlePushData(up)
		case packets.PullData:
			return b.handlePullData(up)
		case packets.TXACK:
			return b.handleTXACK(up)
		default:
			return fmt.Errorf("gateway: unknown packet type: %s", pt)
		}
	})
}

func (b *Backend) handlePullData(up udpPacket) error {
	var p packets.PullDataPacket
	if err := p.UnmarshalBinary(up.data); err != nil {
		return err
	}
	ack := packets.PullACKPacket{
		ProtocolVersion: p.ProtocolVersion,
		RandomToken:     p.RandomToken,
	}
	bytes, err := ack.MarshalBinary()
	if err != nil {
		return errors.Wrap(err, "marshal pull ack packet error")
	}

	err = b.gateways.set(p.GatewayMAC, gateway{
		addr:            up.addr,
		lastSeen:        time.Now().UTC(),
		protocolVersion: p.ProtocolVersion,
	})
	if err != nil {
		return errors.Wrap(err, "set gateway error")
	}

	b.udpSendChan <- udpPacket{
		addr: up.addr,
		data: bytes,
	}
	return nil
}

func (b *Backend) handleTXACK(up udpPacket) error {
	var p packets.TXACKPacket
	if err := p.UnmarshalBinary(up.data); err != nil {
		return err
	}

	if p.Payload != nil && p.Payload.TXPKACK.Error != "" && p.Payload.TXPKACK.Error != "NONE" {
		b.downlinkTXAckChan <- gw.DownlinkTXAck{
			GatewayId: p.GatewayMAC[:],
			Token:     uint32(p.RandomToken),
			Error:     p.Payload.TXPKACK.Error,
		}
	} else {
		b.downlinkTXAckChan <- gw.DownlinkTXAck{
			GatewayId: p.GatewayMAC[:],
			Token:     uint32(p.RandomToken),
		}
	}

	return nil
}

func (b *Backend) handlePushData(up udpPacket) error {
	var p packets.PushDataPacket
	if err := p.UnmarshalBinary(up.data); err != nil {
		return err
	}

	// ack the packet
	ack := packets.PushACKPacket{
		ProtocolVersion: p.ProtocolVersion,
		RandomToken:     p.RandomToken,
	}
	bytes, err := ack.MarshalBinary()
	if err != nil {
		return err
	}
	b.udpSendChan <- udpPacket{
		addr: up.addr,
		data: bytes,
	}

	// gateway stats
	stats, err := p.GetGatewayStats()
	if err != nil {
		return errors.Wrap(err, "get stats error")
	}
	if stats != nil {
		// set gateway ip
		if up.addr.IP.IsLoopback() {
			ip, err := getOutboundIP()
			if err != nil {
				log.WithError(err).Error("gateway: get outbound ip error")
			} else {
				stats.Ip = ip.String()
			}
		} else {
			stats.Ip = up.addr.IP.String()
		}

		b.handleStats(p.GatewayMAC, *stats)
	}

	// uplink frames
	uplinkFrames, err := p.GetUplinkFrames()
	if err != nil {
		return errors.Wrap(err, "get uplink frames error")
	}
	b.handleUplinkFrames(uplinkFrames)

	return nil
}

func (b *Backend) handleStats(gatewayID lorawan.EUI64, stats gw.GatewayStats) {
	// set configuration version, if available
	b.RLock()
	defer b.RUnlock()

	for _, c := range b.configurations {
		if gatewayID == c.MAC {
			stats.ConfigVersion = c.Version
		}
	}

	b.gatewayStatsChan <- stats
}

func (b *Backend) handleUplinkFrames(uplinkFrames []gw.UplinkFrame) error {
	for i := range uplinkFrames {
		b.uplinkFrameChan <- uplinkFrames[i]
	}

	return nil
}

func getOutboundIP() (net.IP, error) {
	// this does not actually connect to 8.8.8.8, unless the connection is
	// used to send UDP frames
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}
