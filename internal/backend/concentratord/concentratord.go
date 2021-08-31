package concentratord

import (
	"context"
	"sync"
	"time"

	"github.com/go-zeromq/zmq4"
	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
)

// Backend implements a ConcentratorD backend.
type Backend struct {
	eventSockCancel   func()
	commandSockCancel func()
	eventSock         zmq4.Socket
	commandSock       zmq4.Socket
	commandMux        sync.Mutex

	// Callback functions for handling events.
	downlinkTxAckFunc           func(gw.DownlinkTXAck)
	gatewayStatsFunc            func(gw.GatewayStats)
	uplinkFrameFunc             func(gw.UplinkFrame)
	rawPacketForwarderEventFunc func(gw.RawPacketForwarderEvent)
	subscribeEventFunc          func(events.Subscribe)

	eventURL   string
	commandURL string

	gatewayID lorawan.EUI64

	crcCheck bool
}

// NewBackend creates a new Backend.
func NewBackend(conf config.Config) (*Backend, error) {
	log.WithFields(log.Fields{
		"event_url":   conf.Backend.Concentratord.EventURL,
		"command_url": conf.Backend.Concentratord.CommandURL,
	}).Info("backend/concentratord: setting up backend")

	b := Backend{
		eventURL:   conf.Backend.Concentratord.EventURL,
		commandURL: conf.Backend.Concentratord.CommandURL,

		crcCheck: conf.Backend.Concentratord.CRCCheck,
	}

	return &b, nil
}

func (b *Backend) dialEventSock() error {
	ctx := context.Background()
	ctx, b.eventSockCancel = context.WithCancel(ctx)

	b.eventSock = zmq4.NewSub(ctx)
	err := b.eventSock.Dial(b.eventURL)
	if err != nil {
		return errors.Wrap(err, "dial event api url error")
	}

	err = b.eventSock.SetOption(zmq4.OptionSubscribe, "")
	if err != nil {
		return errors.Wrap(err, "set event option error")
	}

	log.WithFields(log.Fields{
		"event_url": b.eventURL,
	}).Info("backend/concentratord: connected to event socket")

	return nil
}

func (b *Backend) dialCommandSock() error {
	ctx := context.Background()
	ctx, b.commandSockCancel = context.WithCancel(ctx)

	b.commandSock = zmq4.NewReq(ctx)
	err := b.commandSock.Dial(b.commandURL)
	if err != nil {
		return errors.Wrap(err, "dial command api url error")
	}

	log.WithFields(log.Fields{
		"command_url": b.commandURL,
	}).Info("backend/concentratord: connected to command socket")

	return nil
}

func (b *Backend) dialCommandSockLoop() {
	for {
		if err := b.dialCommandSock(); err != nil {
			log.WithError(err).Error("backend/concentratord: command socket dial error")
			time.Sleep(time.Second)
			continue
		}
		break
	}
}

func (b *Backend) dialEventSockLoop() {
	for {
		if err := b.dialEventSock(); err != nil {
			log.WithError(err).Error("backend/concentratord: event socket dial error")
			time.Sleep(time.Second)
			continue
		}
		break
	}
}

func (b *Backend) getGatewayID() (lorawan.EUI64, error) {
	var gatewayID lorawan.EUI64

	bb, err := b.commandRequest("gateway_id", nil)
	if err != nil {
		return gatewayID, errors.Wrap(err, "request gateway id error")
	}

	copy(gatewayID[:], bb)

	return gatewayID, nil
}

// Start starts the backend.
func (b *Backend) Start() error {
	b.dialEventSockLoop()
	b.dialCommandSockLoop()

	var err error
	b.gatewayID, err = b.getGatewayID()
	if err != nil {
		return errors.Wrap(err, "get gateway id error")
	}

	if b.subscribeEventFunc != nil {
		b.subscribeEventFunc(events.Subscribe{
			Subscribe: true,
			GatewayID: b.gatewayID,
		})
	}

	go b.eventLoop()

	return nil
}

// Stop stops the backend.
func (b *Backend) Stop() error {
	b.eventSock.Close()
	b.commandSock.Close()

	b.eventSockCancel()
	b.commandSockCancel()

	return nil
}

// SetDownlinkTxAckFunc sets the DownlinkTXAck handler func.
func (b *Backend) SetDownlinkTxAckFunc(f func(gw.DownlinkTXAck)) {
	b.downlinkTxAckFunc = f
}

// SetGatewayStatsFunc sets the GatewayStats handler func.
func (b *Backend) SetGatewayStatsFunc(f func(gw.GatewayStats)) {
	b.gatewayStatsFunc = f
}

// SetUplinkFrameFunc sets the UplinkFrame handler func.
func (b *Backend) SetUplinkFrameFunc(f func(gw.UplinkFrame)) {
	b.uplinkFrameFunc = f
}

// SetRawPacketForwarderEventFunc sets the RawPacketForwarderEvent handler func.
func (b *Backend) SetRawPacketForwarderEventFunc(f func(gw.RawPacketForwarderEvent)) {
	b.rawPacketForwarderEventFunc = f
}

// SetSubscribeEventFunc sets the Subscribe handler func.
func (b *Backend) SetSubscribeEventFunc(f func(events.Subscribe)) {
	b.subscribeEventFunc = f
}

// SendDownlinkFrame sends the given downlink frame.
func (b *Backend) SendDownlinkFrame(pl gw.DownlinkFrame) error {
	for i := range pl.GetItems() {
		loRaModInfo := pl.Items[i].GetTxInfo().GetLoraModulationInfo()
		if loRaModInfo != nil {
			loRaModInfo.Bandwidth = loRaModInfo.Bandwidth * 1000
		}
	}

	var downlinkID uuid.UUID
	copy(downlinkID[:], pl.GetDownlinkId())

	log.WithFields(log.Fields{
		"downlink_id": downlinkID,
	}).Info("backend/concentratord: forwarding downlink command")

	bb, err := b.commandRequest("down", &pl)
	if err != nil {
		log.WithError(err).Fatal("backend/concentratord: send downlink command error")
	}
	if len(bb) == 0 {
		return errors.New("no reply receieved, check concentratord logs for error")
	}

	var ack gw.DownlinkTXAck
	if err = proto.Unmarshal(bb, &ack); err != nil {
		return errors.Wrap(err, "protobuf unmarshal error")
	}

	if b.downlinkTxAckFunc != nil {
		b.downlinkTxAckFunc(ack)
	}

	commandCounter("down").Inc()

	return nil
}

// ApplyConfiguration is not implemented.
func (b *Backend) ApplyConfiguration(config gw.GatewayConfiguration) error {
	for i := range config.Channels {
		loRaModConfig := config.Channels[i].GetLoraModulationConfig()
		if loRaModConfig != nil {
			loRaModConfig.Bandwidth = loRaModConfig.Bandwidth * 1000
		}

		fskModConfig := config.Channels[i].GetFskModulationConfig()
		if fskModConfig != nil {
			fskModConfig.Bandwidth = fskModConfig.Bandwidth * 1000
		}
	}

	log.WithFields(log.Fields{
		"version": config.Version,
	}).Info("backend/concentratord: forwarding configuration command")

	_, err := b.commandRequest("config", &config)
	if err != nil {
		log.WithError(err).Fatal("backend/concentratord: send configuration command error")
	}

	commandCounter("config").Inc()

	return nil
}

// RawPacketForwarderCommand is not implemented.
func (b *Backend) RawPacketForwarderCommand(gw.RawPacketForwarderCommand) error {
	return nil
}

func (b *Backend) commandRequest(command string, v proto.Message) ([]byte, error) {
	b.commandMux.Lock()
	defer b.commandMux.Unlock()

	var bb []byte
	var err error

	if v != nil {
		bb, err = proto.Marshal(v)
		if err != nil {
			return nil, errors.Wrap(err, "protobuf marshal error")
		}
	}

	msg := zmq4.NewMsgFrom([]byte(command), bb)
	if err = b.commandSock.SendMulti(msg); err != nil {
		b.commandSockCancel()
		b.dialCommandSock()
		return nil, errors.Wrap(err, "send command request error")
	}

	reply, err := b.commandSock.Recv()
	if err != nil {
		b.commandSockCancel()
		b.dialCommandSock()
		return nil, errors.Wrap(err, "receive command request reply error")
	}

	return reply.Bytes(), nil
}

func (b *Backend) eventLoop() {
	for {
		msg, err := b.eventSock.Recv()
		if err != nil {
			log.WithError(err).Error("backend/concentratord: receive event message error")

			// We need to recover both the event and command sockets.
			func() {
				b.commandMux.Lock()
				defer b.commandMux.Unlock()

				b.eventSockCancel()
				b.commandSockCancel()
				b.dialEventSockLoop()
				b.dialCommandSockLoop()
			}()
			continue
		}

		if len(msg.Frames) == 0 {
			continue
		}

		if len(msg.Frames) != 2 {
			log.WithFields(log.Fields{
				"frame_count": len(msg.Frames),
			}).Error("backend/concentratord: expected 2 frames in event message")
			continue
		}

		switch string(msg.Frames[0]) {
		case "up":
			err = b.handleUplinkFrame(msg.Frames[1])
		case "stats":
			err = b.handleGatewayStats(msg.Frames[1])
		default:
			log.WithFields(log.Fields{
				"event": string(msg.Frames[0]),
			}).Error("backend/concentratord: unexpected event received")
			continue
		}

		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"event": string(msg.Frames[0]),
			}).Error("backend/concentratord: handle event error")
		}

		eventCounter(string(msg.Frames[0])).Inc()
	}
}

func (b *Backend) handleUplinkFrame(bb []byte) error {
	var pl gw.UplinkFrame
	err := proto.Unmarshal(bb, &pl)
	if err != nil {
		return errors.Wrap(err, "protobuf unmarshal error")
	}

	var uplinkID uuid.UUID
	copy(uplinkID[:], pl.GetRxInfo().GetUplinkId())

	if b.crcCheck && pl.GetRxInfo().GetCrcStatus() != gw.CRCStatus_CRC_OK {
		log.WithFields(log.Fields{
			"uplink_id":  uplinkID,
			"crc_status": pl.GetRxInfo().GetCrcStatus(),
		}).Debug("backend/concentratord: ignoring uplink event, CRC is not valid")
		return nil
	}

	// Hz to kHz
	loRaModInfo := pl.GetTxInfo().GetLoraModulationInfo()
	if loRaModInfo != nil {
		loRaModInfo.Bandwidth = loRaModInfo.Bandwidth / 1000
	}

	log.WithFields(log.Fields{
		"uplink_id": uplinkID,
	}).Info("backend/concentratord: uplink event received")

	if b.uplinkFrameFunc != nil {
		b.uplinkFrameFunc(pl)
	}

	return nil
}

func (b *Backend) handleGatewayStats(bb []byte) error {
	var pl gw.GatewayStats
	err := proto.Unmarshal(bb, &pl)
	if err != nil {
		return errors.Wrap(err, "protobuf unmarshal error")
	}

	var statsID uuid.UUID
	copy(statsID[:], pl.GetStatsId())

	// Hz to kHz
	for i := range pl.RxPacketsPerModulation {
		if mod := pl.RxPacketsPerModulation[i].GetModulation(); mod != nil {
			if lora := mod.GetLora(); lora != nil {
				lora.Bandwidth = lora.Bandwidth / 1000
			}
		}
	}

	// Hz to kHz
	for i := range pl.TxPacketsPerModulation {
		if mod := pl.TxPacketsPerModulation[i].GetModulation(); mod != nil {
			if lora := mod.GetLora(); lora != nil {
				lora.Bandwidth = lora.Bandwidth / 1000
			}
		}
	}

	log.WithFields(log.Fields{
		"stats_id": statsID,
	}).Info("backend/concentratord: stats event received")

	if b.gatewayStatsFunc != nil {
		b.gatewayStatsFunc(pl)
	}

	return nil
}
