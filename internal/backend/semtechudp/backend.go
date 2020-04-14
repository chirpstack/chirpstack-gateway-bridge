package semtechudp

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/semtechudp/packets"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/filters"
	"github.com/brocaar/lorawan"
)

// udpPacket represents a raw UDP packet.
type udpPacket struct {
	addr *net.UDPAddr
	data []byte
}

// Backend implements a Semtech packet-forwarder (UDP) gateway backend.
type Backend struct {
	sync.RWMutex

	// tokenMap stores the token to UUID mapping. This should take ~ 1MB of
	// memory. Optionaly this could be optimized by letting keys expire after
	// a given time.
	tokenMap map[uint16][]byte

	downlinkTXAckChan chan gw.DownlinkTXAck
	uplinkFrameChan   chan gw.UplinkFrame
	gatewayStatsChan  chan gw.GatewayStats
	udpSendChan       chan udpPacket

	wg           sync.WaitGroup
	conn         *net.UDPConn
	closed       bool
	gateways     gateways
	fakeRxTime   bool
	skipCRCCheck bool
}

// NewBackend creates a new backend.
func NewBackend(conf config.Config) (*Backend, error) {
	addr, err := net.ResolveUDPAddr("udp", conf.Backend.SemtechUDP.UDPBind)
	if err != nil {
		return nil, errors.Wrap(err, "resolve udp addr error")
	}

	log.WithField("addr", addr).Info("backend/semtechudp: starting gateway udp listener")
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
			gateways:           make(map[lorawan.EUI64]gateway),
			subscribeEventChan: make(chan events.Subscribe),
		},
		fakeRxTime:   conf.Backend.SemtechUDP.FakeRxTime,
		skipCRCCheck: conf.Backend.SemtechUDP.SkipCRCCheck,
		tokenMap:     make(map[uint16][]byte),
	}

	go func() {
		for {
			log.Debug("backend/semtechudp: cleanup gateway registry")
			if err := b.gateways.cleanup(); err != nil {
				log.WithError(err).Error("backend/semtechudp: gateway registry cleanup failed")
			}
			time.Sleep(time.Minute)
		}
	}()

	// Add the waitgroups before the goroutines or a race occurs with closing
	b.wg.Add(2)
	go func() {
		err := b.readPackets()
		if !b.isClosed() {
			log.WithError(err).Error("backend/semtechudp: read udp packets error")
		}
		b.wg.Done()
	}()

	go func() {
		err := b.sendPackets()
		if !b.isClosed() {
			log.WithError(err).Error("backend/semtechudp: send udp packets error")
		}
		b.wg.Done()
	}()

	return b, nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.Lock()
	b.closed = true

	log.Info("backend/semtechudp: closing gateway backend")

	if err := b.conn.Close(); err != nil {
		return errors.Wrap(err, "close udp listener error")
	}

	log.Info("backend/semtechudp: handling last packets")
	close(b.udpSendChan)
	b.Unlock()
	b.wg.Wait()
	return nil
}

// GetDownlinkTXAckChan returns the downlink tx ack channel.
func (b *Backend) GetDownlinkTXAckChan() chan gw.DownlinkTXAck {
	return b.downlinkTXAckChan
}

// GetGatewayStatsChan returns the gateway stats channel.
func (b *Backend) GetGatewayStatsChan() chan gw.GatewayStats {
	return b.gatewayStatsChan
}

// GetUplinkFrameChan returns the uplink frame channel.
func (b *Backend) GetUplinkFrameChan() chan gw.UplinkFrame {
	return b.uplinkFrameChan
}

// GetSubscribeEventChan return the (un)subscribe event channel.
func (b *Backend) GetSubscribeEventChan() chan events.Subscribe {
	return b.gateways.subscribeEventChan
}

// GetRawPacketForwarderEventChan returns the raw packet-forwarder command channel.
func (b *Backend) GetRawPacketForwarderEventChan() chan gw.RawPacketForwarderEvent {
	// not provided by the Semtech packet-forwarder.
	return nil
}

// SendDownlinkFrame sends the given downlink frame to the gateway.
func (b *Backend) SendDownlinkFrame(frame gw.DownlinkFrame) error {
	// mutex is needed in order to write to tokenMap
	b.Lock()
	defer b.Unlock()

	// if Token == 0, generate it in order to be backwards compatible.
	if frame.Token == 0 {
		tokenB := make([]byte, 2)
		if _, err := rand.Read(tokenB); err != nil {
			return errors.Wrap(err, "read random bytes error")
		}
		frame.Token = uint32(binary.BigEndian.Uint16(tokenB))
	}

	// store token to UUID mapping
	b.tokenMap[uint16(frame.Token)] = frame.DownlinkId

	var gatewayID lorawan.EUI64
	copy(gatewayID[:], frame.GetTxInfo().GetGatewayId())

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
		return errors.Wrap(err, "backend/semtechudp: marshal PullRespPacket error")
	}

	b.udpSendChan <- udpPacket{
		data: bytes,
		addr: gw.addr,
	}
	return nil
}

// ApplyConfiguration is not implemented.
func (b *Backend) ApplyConfiguration(config gw.GatewayConfiguration) error {
	return nil
}

// RawPacketForwarderCommand sends the given raw command to the packet-forwarder.
func (b *Backend) RawPacketForwarderCommand(gw.RawPacketForwarderCommand) error {
	return errors.New("raw packet-forwarder command not implemented by Semtech packet-forwarder")
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
				}).Error("backend/semtechudp: could not handle packet")
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
			}).Error("backend/semtechudp: get packet-type error")
			continue
		}

		log.WithFields(log.Fields{
			"addr":             p.addr,
			"type":             pt,
			"protocol_version": p.data[0],
		}).Debug("backend/semtechudp: sending udp packet to gateway")

		_, err = b.conn.WriteToUDP(p.data, p.addr)
		if err != nil {
			log.WithFields(log.Fields{
				"addr":             p.addr,
				"type":             pt,
				"protocol_version": p.data[0],
			}).WithError(err).Error("backend/semtechudp: write to udp error")
		}

		udpWriteCounter(pt.String()).Inc()
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
	}).Debug("backend/semtechudp: received udp packet from gateway")

	udpReadCounter(pt.String()).Inc()

	switch pt {
	case packets.PushData:
		return b.handlePushData(up)
	case packets.PullData:
		return b.handlePullData(up)
	case packets.TXACK:
		return b.handleTXACK(up)
	default:
		return fmt.Errorf("backend/semtechudp: unknown packet type: %s", pt)
	}
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

	downID := b.tokenMap[p.RandomToken]

	if p.Payload != nil && p.Payload.TXPKACK.Error != "" && p.Payload.TXPKACK.Error != "NONE" {
		b.downlinkTXAckChan <- gw.DownlinkTXAck{
			GatewayId:  p.GatewayMAC[:],
			Token:      uint32(p.RandomToken),
			DownlinkId: downID,
			Error:      p.Payload.TXPKACK.Error,
		}
	} else {
		b.downlinkTXAckChan <- gw.DownlinkTXAck{
			GatewayId:  p.GatewayMAC[:],
			Token:      uint32(p.RandomToken),
			DownlinkId: downID,
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
				log.WithError(err).Error("backend/semtechudp: get outbound ip error")
			} else {
				stats.Ip = ip.String()
			}
		} else {
			stats.Ip = up.addr.IP.String()
		}

		b.handleStats(p.GatewayMAC, *stats)
	}

	// uplink frames
	uplinkFrames, err := p.GetUplinkFrames(b.skipCRCCheck, b.fakeRxTime)
	if err != nil {
		return errors.Wrap(err, "get uplink frames error")
	}
	b.handleUplinkFrames(uplinkFrames)

	return nil
}

func (b *Backend) handleStats(gatewayID lorawan.EUI64, stats gw.GatewayStats) {
	b.gatewayStatsChan <- stats
}

func (b *Backend) handleUplinkFrames(uplinkFrames []gw.UplinkFrame) error {
	for i := range uplinkFrames {
		if filters.MatchFilters(uplinkFrames[i].PhyPayload) {
			b.uplinkFrameChan <- uplinkFrames[i]
		} else {
			log.WithFields(log.Fields{
				"data_base64": base64.StdEncoding.EncodeToString(uplinkFrames[i].PhyPayload),
			}).Debug("backend/semtechudp: frame dropped because of configured filters")
		}
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
