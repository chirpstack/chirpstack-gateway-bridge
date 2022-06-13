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
	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/stats"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
	"github.com/brocaar/lorawan/band"
	"github.com/brocaar/lorawan/gps"
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

	caCert  string
	tlsCert string
	tlsKey  string

	server   *http.Server
	ln       net.Listener
	scheme   string
	isClosed bool

	statsInterval    time.Duration
	pingInterval     time.Duration
	timesyncInterval time.Duration
	readTimeout      time.Duration
	writeTimeout     time.Duration

	gateways gateways

	downlinkTxAckFunc           func(gw.DownlinkTXAck)
	uplinkFrameFunc             func(gw.UplinkFrame)
	gatewayStatsFunc            func(gw.GatewayStats)
	rawPacketForwarderEventFunc func(gw.RawPacketForwarderEvent)

	band         band.Band
	region       band.Name
	netIDs       []lorawan.NetID
	joinEUIs     [][2]lorawan.EUI64
	frequencyMin uint32
	frequencyMax uint32
	routerConfig structs.RouterConfig

	// Cache to store diid to UUIDs.
	diidCache *cache.Cache
}

// NewBackend creates a new Backend.
func NewBackend(conf config.Config) (*Backend, error) {
	b := Backend{
		scheme: "ws",

		gateways: gateways{
			gateways: make(map[lorawan.EUI64]*connection),
		},

		caCert:  conf.Backend.BasicStation.CACert,
		tlsCert: conf.Backend.BasicStation.TLSCert,
		tlsKey:  conf.Backend.BasicStation.TLSKey,

		statsInterval:    conf.Backend.BasicStation.StatsInterval,
		pingInterval:     conf.Backend.BasicStation.PingInterval,
		timesyncInterval: conf.Backend.BasicStation.TimesyncInterval,
		readTimeout:      conf.Backend.BasicStation.ReadTimeout,
		writeTimeout:     conf.Backend.BasicStation.WriteTimeout,

		region:       band.Name(conf.Backend.BasicStation.Region),
		frequencyMin: conf.Backend.BasicStation.FrequencyMin,
		frequencyMax: conf.Backend.BasicStation.FrequencyMax,

		diidCache: cache.New(time.Minute, time.Minute),
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
	b.server = &http.Server{
		Handler: mux,
	}

	// if the CA cert is configured, setup client certificate verification.
	if b.caCert != "" {
		rawCACert, err := ioutil.ReadFile(b.caCert)
		if err != nil {
			return nil, errors.Wrap(err, "read ca cert error")
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(rawCACert)

		b.server.TLSConfig = &tls.Config{
			ClientCAs:  caCertPool,
			ClientAuth: tls.RequireAndVerifyClientCert,
		}
	}

	return &b, nil
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
	b.gateways.subscribeEventFunc = f
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

	// Store downlink under DIID in cache
	b.diidCache.SetDefault(fmt.Sprintf("%d", pl.DIID), df)

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

// Start starts the backend.
func (b *Backend) Start() error {
	go func() {
		log.WithFields(log.Fields{
			"bind":     b.ln.Addr(),
			"ca_cert":  b.caCert,
			"tls_cert": b.tlsCert,
			"tls_key":  b.tlsKey,
		}).Info("backend/basicstation: starting websocket listener")

		if b.tlsCert == "" && b.tlsKey == "" && b.caCert == "" {
			// no tls
			if err := b.server.Serve(b.ln); err != nil && !b.isClosed {
				log.WithError(err).Fatal("backend/basicstation: server error")
			}
		} else {
			// tls
			b.scheme = "wss"
			if err := b.server.ServeTLS(b.ln, b.tlsCert, b.tlsKey); err != nil && !b.isClosed {
				log.WithError(err).Fatal("backend/basicstation: server error")
			}
		}
	}()

	return nil
}

// Stop stops the backend.
func (b *Backend) Stop() error {
	b.isClosed = true
	return b.ln.Close()
}

func (b *Backend) handleRouterInfo(r *http.Request, conn *connection) {
	websocketReceiveCounter("router_info").Inc()
	var req structs.RouterInfoRequest

	if err := conn.conn.ReadJSON(&req); err != nil {
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

	conn.Lock()
	defer conn.Unlock()

	conn.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := conn.conn.WriteMessage(websocket.TextMessage, bb); err != nil {
		log.WithError(err).Error("backend/basicstation: websocket send message error")
		return
	}

	log.WithFields(log.Fields{
		"gateway_id":  lorawan.EUI64(req.Router),
		"remote_addr": r.RemoteAddr,
		"router_uri":  resp.URI,
	}).Info("backend/basicstation: router-info request received")
}

func (b *Backend) handleGateway(r *http.Request, conn *connection) {
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
	if err := b.gateways.set(gatewayID, conn); err != nil {
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
		for {
			select {
			case <-statsTicker.C:
				id, err := uuid.NewV4()
				if err != nil {
					log.WithError(err).Error("backend/basicstation: new uuid error")
					continue
				}

				stats := conn.stats.ExportStats()
				stats.GatewayId = gatewayID[:]
				stats.Time = ptypes.TimestampNow()
				stats.StatsId = id[:]

				if b.gatewayStatsFunc != nil {
					b.gatewayStatsFunc(stats)
				}
			case <-done:
				return
			}
		}

	}()

	// receive data
	for {
		mt, msg, err := conn.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithField("gateway_id", gatewayID).WithError(err).Error("backend/basicstation: read message error")
			}
			return
		}

		// reset the read deadline as the Basic Station doesn't respond to PONG messages (yet)
		conn.conn.SetReadDeadline(time.Now().Add(b.readTimeout))

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
			b.sendTimesyncRequest(gatewayID, pl.RadioMetaData.UpInfo)
		case structs.JoinRequestMessage:
			// handle join-request
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
			b.sendTimesyncRequest(gatewayID, pl.RadioMetaData.UpInfo)
		case structs.ProprietaryDataFrameMessage:
			// handle proprietary uplink
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
			b.sendTimesyncRequest(gatewayID, pl.RadioMetaData.UpInfo)
		case structs.DownlinkTransmittedMessage:
			// handle downlink transmitted
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
		case structs.TimeSyncMessage:
			// handle time sync request
			var pl structs.TimeSyncRequest
			if err := json.Unmarshal(msg, &pl); err != nil {
				log.WithError(err).WithFields(log.Fields{
					"message_type": msgType,
					"gateway_id":   gatewayID,
					"payload":      string(msg),
				}).Error("backend/basicstation: unmarshal json message error")
				continue
			}
			b.handleTimeSync(gatewayID, pl)
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

	if conn, err := b.gateways.get(gatewayID); err == nil {
		conn.stats.CountUplink(&uplinkFrame)
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"uplink_id":  uplinkID,
	}).Info("backend/basicstation: join-request received")

	if b.uplinkFrameFunc != nil {
		b.uplinkFrameFunc(uplinkFrame)
	}
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

	if conn, err := b.gateways.get(gatewayID); err == nil {
		conn.stats.CountUplink(&uplinkFrame)
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"uplink_id":  uplinkID,
	}).Info("backend/basicstation: proprietary uplink frame received")

	if b.uplinkFrameFunc != nil {
		b.uplinkFrameFunc(uplinkFrame)
	}
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
		pl := v.(gw.DownlinkFrame)
		txack.DownlinkId = pl.DownlinkId

		if conn, err := b.gateways.get(gatewayID); err == nil {
			conn.stats.CountDownlink(&pl, &txack)
		}
	}

	var downID uuid.UUID
	copy(downID[:], txack.GetDownlinkId())

	log.WithFields(log.Fields{
		"gateway_id":  gatewayID,
		"downlink_id": downID,
	}).Info("backend/basicstation: downlink transmitted message received")

	if b.downlinkTxAckFunc != nil {
		b.downlinkTxAckFunc(txack)
	}
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

	// count metrics
	if conn, err := b.gateways.get(gatewayID); err == nil {
		conn.stats.CountUplink(&uplinkFrame)
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"uplink_id":  uplinkID,
	}).Info("backend/basicstation: uplink frame received")

	if b.uplinkFrameFunc != nil {
		b.uplinkFrameFunc(uplinkFrame)
	}
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

	if b.rawPacketForwarderEventFunc != nil {
		b.rawPacketForwarderEventFunc(rawEvent)
	}
}

func (b *Backend) handleTimeSync(gatewayID lorawan.EUI64, v structs.TimeSyncRequest) {
	resp := structs.TimeSyncResponse{
		MessageType: structs.TimeSyncMessage,
		TxTime:      v.TxTime,
		GPSTime:     int64(gps.Time(time.Now()).TimeSinceGPSEpoch() / time.Microsecond),
	}
	if err := b.sendToGateway(gatewayID, &resp); err != nil {
		log.WithError(err).Error("backend/basicstation: send to gateway error")
		return
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"txtime":     resp.TxTime,
		"gpstime":    resp.GPSTime,
	}).Info("backend/basicstation: timesync response sent to gateway")
}

func (b *Backend) sendToGateway(gatewayID lorawan.EUI64, v interface{}) error {
	conn, err := b.gateways.get(gatewayID)
	if err != nil {
		return errors.Wrap(err, "get gateway error")
	}

	conn.Lock()
	defer conn.Unlock()

	bb, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshal json error")
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"message":    string(bb),
	}).Debug("sending message to gateway")

	conn.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := conn.conn.WriteMessage(websocket.TextMessage, bb); err != nil {
		return errors.Wrap(err, "send message to gateway error")
	}

	return nil
}

func (b *Backend) sendRawToGateway(gatewayID lorawan.EUI64, messageType int, data []byte) error {
	conn, err := b.gateways.get(gatewayID)
	if err != nil {
		return errors.Wrap(err, "get gateway error")
	}

	conn.Lock()
	defer conn.Unlock()

	conn.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := conn.conn.WriteMessage(messageType, data); err != nil {
		return errors.Wrap(err, "send message to gateway error")
	}

	return nil
}

func (b *Backend) websocketWrap(handler func(*http.Request, *connection), w http.ResponseWriter, r *http.Request) {
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

	// Wrap the conn inside a gateway struct, so that we can lock it when writing
	// data.
	c := connection{conn: conn, stats: stats.NewCollector()}

	go func() {
		for {
			select {
			case <-ticker.C:
				c.Lock()
				websocketPingPongCounter("ping").Inc()
				c.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.WithError(err).Error("backend/basicstation: send ping message error")
					c.conn.Close()
				}
				c.Unlock()
			case <-done:
				return
			}
		}
	}()

	handler(r, &c)
	done <- struct{}{}
}

func (b *Backend) sendTimesyncRequest(gatewayID lorawan.EUI64, upInfo structs.RadioMetaDataUpInfo) {
	// Nothing to do
	if b.timesyncInterval == 0 {
		return
	}

	lastTimesync, err := b.gateways.getLastTimesync(gatewayID)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: get last timesync timestamp error")
		return
	}

	// Interval has not been reached yet
	if lastTimesync.Add(b.timesyncInterval).After(time.Now()) {
		return
	}

	// Set last timesync
	if err := b.gateways.setLastTimesync(gatewayID, time.Now()); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"gateway_id": gatewayID,
		}).Error("backend/basicstation: set last timesync timestamp error")
		return
	}

	timesync := structs.TimeSyncGPSTimeTransfer{
		MessageType: structs.TimeSyncMessage,
		XTime:       upInfo.XTime,
		GPSTime:     int64(gps.Time(time.Now()).TimeSinceGPSEpoch() / time.Microsecond),
	}

	if err := b.sendToGateway(gatewayID, &timesync); err != nil {
		log.WithError(err).Error("backend/basicstation: send to gateway error")
		return
	}

	log.WithFields(log.Fields{
		"gateway_id": gatewayID,
		"xtime":      timesync.XTime,
		"gpstime":    timesync.GPSTime,
	}).Info("backend/basicstation: timesync request sent to gateway")
}
