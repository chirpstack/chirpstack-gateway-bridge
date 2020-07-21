package basicstation

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/websocket"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-api/go/v3/gw"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/basicstation/structs"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/events"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
)

// websocket upgrade parameters
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// Backend implements a Basic Station backend.
type Backend struct {
	sync.RWMutex

	ln       net.Listener
	scheme   string
	isClosed bool

	statsInterval time.Duration
	pingInterval  time.Duration
	readTimeout   time.Duration
	writeTimeout  time.Duration

	gateways gateways

	downlinkTXAckChan           chan gw.DownlinkTXAck
	uplinkFrameChan             chan gw.UplinkFrame
	gatewayStatsChan            chan gw.GatewayStats
	rawPacketForwarderEventChan chan gw.RawPacketForwarderEvent

	band         band.Band
	region       band.Name
	netIDs       []lorawan.NetID
	joinEUIs     [][2]lorawan.EUI64
	frequencyMin uint32
	frequencyMax uint32
	routerConfig structs.RouterConfig

	// Cache to store stats.
	statsCache *cache.Cache

	// Cache to store diid to UUIDs.
	diidCache *cache.Cache
}

// NewBackend creates a new Backend.
func NewBackend(conf config.Config) (*Backend, error) {
	b := Backend{
		scheme: "ws",

		gateways: gateways{
			gateways:           make(map[lorawan.EUI64]gateway),
			subscribeEventChan: make(chan events.Subscribe),
		},

		downlinkTXAckChan:           make(chan gw.DownlinkTXAck),
		uplinkFrameChan:             make(chan gw.UplinkFrame),
		gatewayStatsChan:            make(chan gw.GatewayStats),
		rawPacketForwarderEventChan: make(chan gw.RawPacketForwarderEvent),

		statsInterval: conf.Backend.BasicStation.StatsInterval,
		pingInterval:  conf.Backend.BasicStation.PingInterval,
		readTimeout:   conf.Backend.BasicStation.ReadTimeout,
		writeTimeout:  conf.Backend.BasicStation.WriteTimeout,

		region:       band.Name(conf.Backend.BasicStation.Region),
		frequencyMin: conf.Backend.BasicStation.FrequencyMin,
		frequencyMax: conf.Backend.BasicStation.FrequencyMax,

		diidCache:  cache.New(time.Minute, time.Minute),
		statsCache: cache.New(conf.Backend.BasicStation.StatsInterval*2, conf.Backend.BasicStation.StatsInterval*2),
	}

	for _, n := range conf.Filters.NetIDs {
		var netID lorawan.NetID
		if err := netID.UnmarshalText([]byte(n)); err != nil {
			return nil, errors.Wrap(err, "unmarshal netid error")
		}
		b.netIDs = append(b.netIDs, netID)
	}

	for _, set := range conf.Filters.JoinEUIs {
		var joinEUIs [2]lorawan.EUI64
		for i, s := range set {
			var eui lorawan.EUI64
			if err := eui.UnmarshalText([]byte(s)); err != nil {
				return nil, errors.Wrap(err, "unmarshal joineui error")
			}
			joinEUIs[i] = eui
		}
		b.joinEUIs = append(b.joinEUIs, joinEUIs)
	}

	var err error
	b.band, err = band.GetConfig(b.region, false, lorawan.DwellTimeNoLimit)
	if err != nil {
		return nil, errors.Wrap(err, "get band config error")
	}

	b.routerConfig, err = structs.GetRouterConfig(b.region, b.netIDs, b.joinEUIs, b.frequencyMin, b.frequencyMax, conf.Backend.BasicStation.Concentrators)
	if err != nil {
		return nil, errors.Wrap(err, "get router config error")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/router-info", func(w http.ResponseWriter, r *http.Request) {
		b.websocketWrap(b.handleRouterInfo, w, r)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		connectCounter().Inc()
		b.websocketWrap(b.handleGateway, w, r)
		disconnectCounter().Inc()
	})

	// using net.Listen makes it easier to test as we can bind to ":0" and
	// then read back the Addr to find the assigned (random) port.
	b.ln, err = net.Listen("tcp", conf.Backend.BasicStation.Bind)
	if err != nil {
		return nil, errors.Wrap(err, "create listener error")
	}

	// init HTTP server
	server := &http.Server{
		Handler: mux,
	}

	// if the CA cert is configured, setup client certificate verification.
	if conf.Backend.BasicStation.CACert != "" {
		rawCACert, err := ioutil.ReadFile(conf.Backend.BasicStation.CACert)
		if err != nil {
			return nil, errors.Wrap(err, "read ca cert error")
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(rawCACert)

		server.TLSConfig = &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
	}

	go func() {
		log.WithFields(log.Fields{
			"bind":     b.ln.Addr(),
			"tls_cert": conf.Backend.BasicStation.TLSCert,
			"tls_key":  conf.Backend.BasicStation.TLSKey,
			"ca_cert":  conf.Backend.BasicStation.CACert,
		}).Info("backend/basicstation: starting websocket listener")

		if conf.Backend.BasicStation.TLSCert == "" && conf.Backend.BasicStation.TLSKey == "" && conf.Backend.BasicStation.CACert == "" {
			// no tls
			if err := server.Serve(b.ln); err != nil && !b.isClosed {
				log.WithError(err).Fatal("backend/basicstation: server error")
			}
		} else {
			// tls
			b.scheme = "wss"
			if err := server.ServeTLS(b.ln, conf.Backend.BasicStation.TLSCert, conf.Backend.BasicStation.TLSKey); err != nil && !b.isClosed {
				log.WithError(err).Fatal("backend/basicstation: server error")
			}
		}
	}()

	return &b, nil
}

// GetDownlinkTXAckChan returns the channel for downlink tx acknowledgements.
func (b *Backend) GetDownlinkTXAckChan() chan gw.DownlinkTXAck {
	return b.downlinkTXAckChan
}

// GetGatewayStatsChan returns the channel for gateway statistics.
func (b *Backend) GetGatewayStatsChan() chan gw.GatewayStats {
	return b.gatewayStatsChan
}

// GetUplinkFrameChan returns the channel for received uplinks.
func (b *Backend) GetUplinkFrameChan() chan gw.UplinkFrame {
	return b.uplinkFrameChan
}

// GetSubscribeEventChan returns the channel for the (un)subscribe events.
func (b *Backend) GetSubscribeEventChan() chan events.Subscribe {
	return b.gateways.subscribeEventChan
}

// GetRawPacketForwarderEventChan returns the raw packet-forwarder command channel.
func (b *Backend) GetRawPacketForwarderEventChan() chan gw.RawPacketForwarderEvent {
	return b.rawPacketForwarderEventChan
}

// SendDownlinkFrame sends the given downlink frame.
func (b *Backend) SendDownlinkFrame(df gw.DownlinkFrame) error {
	b.Lock()
	defer b.Unlock()

	// for backwards compatibility
	if df.Token == 0 {
		tokenB := make([]byte, 2)
		if _, err := rand.Read(tokenB); err != nil {
			return errors.Wrap(err, "read random bytes error")
		}

		df.Token = uint32(binary.BigEndian.Uint16(tokenB))
	}

	pl, err := structs.DownlinkFrameFromProto(b.band, df)
	if err != nil {
		return errors.Wrap(err, "downlink frame from proto error")
	}

	var gatewayID lorawan.EUI64
	var downID uuid.UUID
	copy(gatewayID[:], df.GetGatewayId())
	copy(downID[:], df.GetDownlinkId())

	b.incrementTxStats(gatewayID)

	// store token to UUID mapping
	b.diidCache.SetDefault(fmt.Sprintf("%d", df.Token), df.GetDownlinkId())

	websocketSendCounter("dnmsg").Inc()
	if err := b.sendToGateway(gatewayID, pl); err != nil {
		return errors.Wrap(err, "send to gateway error")
	}

	log.WithFields(log.Fields{
		"gateway_id":  gatewayID,
		"downlink_id": downID,
	}).Info("backend/basicstation: downlink-frame message sent to gateway")

	return nil
}

// ApplyConfiguration is not implemented.
func (b *Backend) ApplyConfiguration(gwConfig gw.GatewayConfiguration) error {
	return nil
}

// RawPacketForwarderCommand sends the given raw command to the packet-forwarder.
func (b *Backend) RawPacketForwarderCommand(pl gw.RawPacketForwarderCommand) error {
	var gatewayID lorawan.EUI64
	var rawID uuid.UUID

	copy(gatewayID[:], pl.GatewayId)
	copy(rawID[:], pl.RawId)

	if len(pl.Payload) == 0 {
		return errors.New("raw packet-forwarder command payload is empty")
	}

	mt := websocket.BinaryMessage
	if strings.HasPrefix(string(pl.Payload), "{") {
		mt = websocket.TextMessage
	}

	websocketSendCounter("raw").Inc()
	if err := b.sendRawToGateway(gatewayID, mt, pl.Payload); err != nil {
		return errors.Wrap(err, "send raw packet-forwarder command to gateway error")
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"raw_id":     rawID,
	}).Info("backend/basicstation: raw packet-forwarder command sent to gateway")

	return nil
}

// Close closes the backend.
func (b *Backend) Close() error {
	b.isClosed = true
	return b.ln.Close()
}

func (b *Backend) handleRouterInfo(r *http.Request, c *websocket.Conn) {
	websocketReceiveCounter("router_info").Inc()
	var req structs.RouterInfoRequest

	if err := c.ReadJSON(&req); err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.WithError(err).Error("backend/basicstation: read message error")
		}
		return
	}

	resp := structs.RouterInfoResponse{
		Router: req.Router,
		Muxs:   req.Router,
		URI:    fmt.Sprintf("%s://%s/gateway/%s", b.scheme, r.Host, lorawan.EUI64(req.Router)),
	}

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		var cn lorawan.EUI64

		if err := cn.UnmarshalText([]byte(r.TLS.PeerCertificates[0].Subject.CommonName)); err != nil || cn != lorawan.EUI64(req.Router) {
			resp.URI = ""
			resp.Error = fmt.Sprintf("certificate CommonName %s does not match router %s",
				r.TLS.PeerCertificates[0].Subject.CommonName, lorawan.EUI64(req.Router))
		}
	}

	bb, err := json.Marshal(resp)
	if err != nil {
		log.WithError(err).Error("backend/basicstation: marshal json error")
		return
	}

	c.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := c.WriteMessage(websocket.TextMessage, bb); err != nil {
		log.WithError(err).Error("backend/basicstation: websocket send message error")
		return
	}

	log.WithFields(log.Fields{
		"gateway_id":  lorawan.EUI64(req.Router),
		"remote_addr": r.RemoteAddr,
		"router_uri":  resp.URI,
	}).Info("backend/basicstation: router-info request received")
}

func (b *Backend) handleGateway(r *http.Request, c *websocket.Conn) {
	// get the gateway id from the url
	urlParts := strings.Split(r.URL.Path, "/")
	if len(urlParts) < 2 {
		log.WithField("url", r.URL.Path).Error("backend/basicstation: unable to read gateway id from url")
		return
	}

	var gatewayID lorawan.EUI64
	if err := gatewayID.UnmarshalText([]byte(urlParts[len(urlParts)-1])); err != nil {
		log.WithError(err).Error("backend/basicstation: parse gateway id error")
		return
	}

	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		var cn lorawan.EUI64
		if err := cn.UnmarshalText([]byte(r.TLS.PeerCertificates[0].Subject.CommonName)); err != nil || cn != gatewayID {
			log.WithFields(log.Fields{
				"gateway_id":  gatewayID,
				"common_name": r.TLS.PeerCertificates[0].Subject.CommonName,
			}).Error("backend/basicstation: CommonName verification failed")
			return
		}
	}

	// make sure we're not overwriting an existing connection
	_, err := b.gateways.get(gatewayID)
	if err == nil {
		log.WithField("gateway_id", gatewayID).Error("backend/basicstation: connection with same gateway id already exists")
		return
	}

	// set the gateway connection
	if err := b.gateways.set(gatewayID, gateway{conn: c}); err != nil {
		log.WithError(err).WithField("gateway_id", gatewayID).Error("backend/basicstation: set gateway error")
	}
	log.WithFields(log.Fields{
		"gateway_id":  gatewayID,
		"remote_addr": r.RemoteAddr,
	}).Info("backend/basicstation: gateway connected")

	done := make(chan struct{})

	// remove the gateway on return
	defer func() {
		done <- struct{}{}
		b.gateways.remove(gatewayID)
		log.WithFields(log.Fields{
			"gateway_id":  gatewayID,
			"remote_addr": r.RemoteAddr,
		}).Info("backend/basicstation: gateway disconnected")
	}()

	statsTicker := time.NewTicker(b.statsInterval)
	defer statsTicker.Stop()

	// stats publishing loop
	go func() {
		gwIDStr := gatewayID.String()

		for {
			select {
			case <-statsTicker.C:
				id, err := uuid.NewV4()
				if err != nil {
					log.WithError(err).Error("backend/basicstation: new uuid error")
					continue
				}

				var rx, rxOK, tx, txOK uint32
				if v, ok := b.statsCache.Get(gwIDStr + ":rx"); ok {
					rx = v.(uint32)
				}
				if v, ok := b.statsCache.Get(gwIDStr + ":rxOK"); ok {
					rxOK = v.(uint32)
				}
				if v, ok := b.statsCache.Get(gwIDStr + ":tx"); ok {
					tx = v.(uint32)
				}
				if v, ok := b.statsCache.Get(gwIDStr + ":txOK"); ok {
					txOK = v.(uint32)
				}

				b.statsCache.Delete(gwIDStr + ":rx")
				b.statsCache.Delete(gwIDStr + ":rxOK")
				b.statsCache.Delete(gwIDStr + ":tx")
				b.statsCache.Delete(gwIDStr + ":txOK")

				b.gatewayStatsChan <- gw.GatewayStats{
					GatewayId:           gatewayID[:],
					Time:                ptypes.TimestampNow(),
					StatsId:             id[:],
					RxPacketsReceived:   rx,
					RxPacketsReceivedOk: rxOK,
					TxPacketsReceived:   tx,
					TxPacketsEmitted:    txOK,
				}
			case <-done:
				return
			}
		}

	}()

	// receive data
	for {
		mt, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithField("gateway_id", gatewayID).WithError(err).Error("backend/basicstation: read message error")
			}
			return
		}

		// reset the read deadline as the Basic Station doesn't respond to PONG messages (yet)
		c.SetReadDeadline(time.Now().Add(b.readTimeout))

		if mt == websocket.BinaryMessage {
			log.WithFields(log.Fields{
				"gateway_id":     gatewayID,
				"message_base64": base64.StdEncoding.EncodeToString(msg),
			}).Debug("backend/basicstation: binary message received")

			b.handleRawPacketForwarderEvent(gatewayID, msg)
			continue
		}

		log.WithFields(log.Fields{
			"gateway_id": gatewayID,
			"message":    string(msg),
		}).Debug("backend/basicstation: message received")

		// get message-type
		msgType, err := structs.GetMessageType(msg)
		if err != nil {
			log.WithFields(log.Fields{
				"gateway_id": gatewayID,
				"payload":    string(msg),
			}).WithError(err).Error("backend/basicstation: get message-type error")
			continue
		}

		websocketReceiveCounter(string(msgType)).Inc()

		// handle message-type
		switch msgType {
		case structs.VersionMessage:
			// handle version
			var pl structs.Version
			if err := json.Unmarshal(msg, &pl); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_type": msgType,
					"gateway_id":   gatewayID,
					"payload":      string(msg),
				}).Error("backend/basicstation: unmarshal json message error")
				continue
			}
			b.handleVersion(gatewayID, pl)
		case structs.UplinkDataFrameMessage:
			// handle uplink
			b.incrementRxStats(gatewayID)
			var pl structs.UplinkDataFrame
			if err := json.Unmarshal(msg, &pl); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_type": msgType,
					"gateway_id":   gatewayID,
					"payload":      string(msg),
				}).Error("backend/basicstation: unmarshal json message error")
				continue
			}
			b.handleUplinkDataFrame(gatewayID, pl)
		case structs.JoinRequestMessage:
			// handle join-request
			b.incrementRxStats(gatewayID)
			var pl structs.JoinRequest
			if err := json.Unmarshal(msg, &pl); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_type": msgType,
					"gateway_id":   gatewayID,
					"payload":      string(msg),
				}).Error("backend/basicstation: unmarshal json message error")
				continue
			}
			b.handleJoinRequest(gatewayID, pl)
		case structs.ProprietaryDataFrameMessage:
			// handle proprietary uplink
			b.incrementRxStats(gatewayID)
			var pl structs.UplinkProprietaryFrame
			if err := json.Unmarshal(msg, &pl); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_type": msgType,
					"gateway_id":   gatewayID,
					"payload":      string(msg),
				}).Error("backend/basicstation: unmarshal json message error")
				continue
			}
			b.handleProprietaryDataFrame(gatewayID, pl)
		case structs.DownlinkTransmittedMessage:
			// handle downlink transmitted
			b.incrementTxOkStats(gatewayID)
			var pl structs.DownlinkTransmitted
			if err := json.Unmarshal(msg, &pl); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_type": msgType,
					"gateway_id":   gatewayID,
					"payload":      string(msg),
				}).Error("backend/basicstation: unmarshal json message error")
				continue
			}
			b.handleDownlinkTransmittedMessage(gatewayID, pl)
		default:
			b.handleRawPacketForwarderEvent(gatewayID, msg)
		}
	}
}

func (b *Backend) handleVersion(gatewayID lorawan.EUI64, pl structs.Version) {
	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"station":    pl.Station,
		"firmware":   pl.Firmware,
		"package":    pl.Package,
		"model":      pl.Model,
		"protocol":   pl.Protocol,
		// "features":   pl.Features,
	}).Info("backend/basicstation: gateway version received")

	websocketSendCounter("router_config").Inc()
	if err := b.sendToGateway(gatewayID, b.routerConfig); err != nil {
		log.WithError(err).Error("backend/basicstation: send to gateway error")
		return
	}

	log.WithField("gateway_id", gatewayID).Info("backend/basicstation: router-config message sent to gateway")
}

func (b *Backend) handleJoinRequest(gatewayID lorawan.EUI64, v structs.JoinRequest) {
	uplinkFrame, err := structs.JoinRequestToProto(b.band, gatewayID, v)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: error converting join-request to protobuf message")
		return
	}

	// set uplink id
	uplinkID, err := uuid.NewV4()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: get random uplink id error")
		return
	}
	uplinkFrame.RxInfo.UplinkId = uplinkID[:]

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"uplink_id":  uplinkID,
	}).Info("backend/basicstation: join-request received")

	b.uplinkFrameChan <- uplinkFrame
}

func (b *Backend) handleProprietaryDataFrame(gatewayID lorawan.EUI64, v structs.UplinkProprietaryFrame) {
	uplinkFrame, err := structs.UplinkProprietaryFrameToProto(b.band, gatewayID, v)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: error converting proprietary uplink to protobuf message")
		return
	}

	// set uplink id
	uplinkID, err := uuid.NewV4()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: get random uplink id error")
		return
	}
	uplinkFrame.RxInfo.UplinkId = uplinkID[:]

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"uplink_id":  uplinkID,
	}).Info("backend/basicstation: proprietary uplink frame received")

	b.uplinkFrameChan <- uplinkFrame
}

func (b *Backend) handleDownlinkTransmittedMessage(gatewayID lorawan.EUI64, v structs.DownlinkTransmitted) {
	b.RLock()
	defer b.RUnlock()

	txack, err := structs.DownlinkTransmittedToProto(gatewayID, v)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: error converting downlink transmitted to protobuf message")
		return
	}

	if v, ok := b.diidCache.Get(fmt.Sprintf("%d", v.DIID)); ok {
		txack.DownlinkId = v.([]byte)
	}

	var downID uuid.UUID
	copy(downID[:], txack.GetDownlinkId())

	log.WithFields(log.Fields{
		"gateway_id":  gatewayID,
		"downlink_id": downID,
	}).Info("backend/basicstation: downlink transmitted message received")

	b.downlinkTXAckChan <- txack
}

func (b *Backend) handleUplinkDataFrame(gatewayID lorawan.EUI64, v structs.UplinkDataFrame) {
	uplinkFrame, err := structs.UplinkDataFrameToProto(b.band, gatewayID, v)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: error converting uplink frame to protobuf message")
		return
	}

	// set uplink id
	uplinkID, err := uuid.NewV4()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: get random uplink id error")
		return
	}
	uplinkFrame.RxInfo.UplinkId = uplinkID[:]

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"uplink_id":  uplinkID,
	}).Info("backend/basicstation: uplink frame received")

	b.uplinkFrameChan <- uplinkFrame
}

func (b *Backend) handleRawPacketForwarderEvent(gatewayID lorawan.EUI64, pl []byte) {
	rawID, err := uuid.NewV4()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: get random raw id error")
		return
	}

	rawEvent := gw.RawPacketForwarderEvent{
		GatewayId: gatewayID[:],
		RawId:     rawID[:],
		Payload:   pl,
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"raw_id":     rawID,
	}).Info("backend/basicstation: raw packet-forwarder event received")

	b.rawPacketForwarderEventChan <- rawEvent
}

func (b *Backend) sendToGateway(gatewayID lorawan.EUI64, v interface{}) error {
	gw, err := b.gateways.get(gatewayID)
	if err != nil {
		return errors.Wrap(err, "get gateway error")
	}

	bb, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshal json error")
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"message":    string(bb),
	}).Debug("sending message to gateway")

	gw.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := gw.conn.WriteMessage(websocket.TextMessage, bb); err != nil {
		return errors.Wrap(err, "send message to gateway error")
	}

	return nil
}

func (b *Backend) sendRawToGateway(gatewayID lorawan.EUI64, messageType int, data []byte) error {
	gw, err := b.gateways.get(gatewayID)
	if err != nil {
		return errors.Wrap(err, "get gateway error")
	}

	gw.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := gw.conn.WriteMessage(messageType, data); err != nil {
		return errors.Wrap(err, "send message to gateway error")
	}

	return nil
}

func (b *Backend) websocketWrap(handler func(*http.Request, *websocket.Conn), w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.WithError(err).Error("backend/basicstation: websocket upgrade error")
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(b.readTimeout))
	conn.SetPongHandler(func(string) error {
		websocketPingPongCounter("pong").Inc()
		conn.SetReadDeadline(time.Now().Add(b.readTimeout))
		return nil
	})

	ticker := time.NewTicker(b.pingInterval)
	defer ticker.Stop()
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				websocketPingPongCounter("ping").Inc()
				conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.WithError(err).Error("backend/basicstation: send ping message error")
					conn.Close()
				}
			case <-done:
				return
			}
		}
	}()

	handler(r, conn)
	done <- struct{}{}
}

func (b *Backend) incrementRxStats(id lorawan.EUI64) {
	idStr := id.String()

	if _, err := b.statsCache.IncrementUint32(idStr+":rx", uint32(1)); err != nil {
		b.statsCache.SetDefault(idStr+":rx", uint32(1))
	}

	if _, err := b.statsCache.IncrementUint32(idStr+":rxOK", uint32(1)); err != nil {
		b.statsCache.SetDefault(idStr+":rxOK", uint32(1))
	}
}

func (b *Backend) incrementTxOkStats(id lorawan.EUI64) {
	idStr := id.String()

	if _, err := b.statsCache.IncrementUint32(idStr+"txOK", uint32(1)); err != nil {
		b.statsCache.SetDefault(idStr+":txOK", uint32(1))
	}
}

func (b *Backend) incrementTxStats(id lorawan.EUI64) {
	idStr := id.String()

	if _, err := b.statsCache.IncrementUint32(idStr+"tx", uint32(1)); err != nil {
		b.statsCache.SetDefault(idStr+":tx", uint32(1))
	}
}
