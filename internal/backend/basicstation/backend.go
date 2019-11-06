package basicstation

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/backend/basicstation/structs"
	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/chirpstack-api/go/gw"
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

	pingInterval time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration

	gateways gateways

	downlinkTXAckChan chan gw.DownlinkTXAck
	uplinkFrameChan   chan gw.UplinkFrame
	gatewayStatsChan  chan gw.GatewayStats

	band         band.Band
	region       band.Name
	netIDs       []lorawan.NetID
	joinEUIs     [][2]lorawan.EUI64
	frequencyMin uint32
	frequencyMax uint32
	routerConfig *structs.RouterConfig

	// diidMap stores the mapping of diid to UUIDs. This should take ~ 1MB of
	// memory. Optionaly this could be optimized by letting keys expire after
	// a given time.
	diidMap map[uint16][]byte
}

// NewBackend creates a new Backend.
func NewBackend(conf config.Config) (*Backend, error) {
	b := Backend{
		scheme: "ws",

		gateways: gateways{
			gateways:       make(map[lorawan.EUI64]gateway),
			connectChan:    make(chan lorawan.EUI64),
			disconnectChan: make(chan lorawan.EUI64),
		},

		downlinkTXAckChan: make(chan gw.DownlinkTXAck),
		uplinkFrameChan:   make(chan gw.UplinkFrame),
		gatewayStatsChan:  make(chan gw.GatewayStats),

		pingInterval: conf.Backend.BasicStation.PingInterval,
		readTimeout:  conf.Backend.BasicStation.ReadTimeout,
		writeTimeout: conf.Backend.BasicStation.WriteTimeout,

		region:       band.Name(conf.Backend.BasicStation.Region),
		frequencyMin: conf.Backend.BasicStation.FrequencyMin,
		frequencyMax: conf.Backend.BasicStation.FrequencyMax,

		diidMap: make(map[uint16][]byte),
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

	if len(conf.Backend.BasicStation.Concentrators) != 0 {
		conf, err := structs.GetRouterConfig(b.region, b.netIDs, b.joinEUIs, b.frequencyMin, b.frequencyMax, conf.Backend.BasicStation.Concentrators)
		if err != nil {
			return nil, errors.Wrap(err, "get router config error")
		}

		b.routerConfig = &conf
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

func (b *Backend) GetDownlinkTXAckChan() chan gw.DownlinkTXAck {
	return b.downlinkTXAckChan
}

func (b *Backend) GetGatewayStatsChan() chan gw.GatewayStats {
	return b.gatewayStatsChan
}

func (b *Backend) GetUplinkFrameChan() chan gw.UplinkFrame {
	return b.uplinkFrameChan
}

func (b *Backend) GetConnectChan() chan lorawan.EUI64 {
	return b.gateways.connectChan
}

func (b *Backend) GetDisconnectChan() chan lorawan.EUI64 {
	return b.gateways.disconnectChan
}

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
	copy(gatewayID[:], df.GetTxInfo().GetGatewayId())
	copy(downID[:], df.GetDownlinkId())

	// store token to UUID mapping
	b.diidMap[uint16(df.Token)] = df.GetDownlinkId()

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

func (b *Backend) ApplyConfiguration(gwConfig gw.GatewayConfiguration) error {
	rc, err := structs.GetRouterConfigOld(b.region, b.netIDs, b.joinEUIs, b.frequencyMin, b.frequencyMax, gwConfig)
	if err != nil {
		return errors.Wrap(err, "get router config error")
	}

	var gatewayID lorawan.EUI64
	copy(gatewayID[:], gwConfig.GetGatewayId())

	websocketSendCounter("router_config").Inc()
	if err := b.sendToGateway(gatewayID, rc); err != nil {
		return errors.Wrap(err, "send router config to gateway error")
	}

	log.WithField("gateway_id", gatewayID).Info("backend/basicstation: router-config message sent to gateway")

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

	c.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := c.WriteJSON(resp); err != nil {
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

	// remove the gateway on return
	defer func() {
		b.gateways.remove(gatewayID)
		log.WithFields(log.Fields{
			"gateway_id":  gatewayID,
			"remote_addr": r.RemoteAddr,
		}).Info("backend/basicstation: gateway disconnected")
	}()

	// receive data
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithField("gateway_id", gatewayID).WithError(err).Error("backend/basicstation: read message error")
			}
			return
		}

		// reset the read deadline as the Basic Station doesn't respond to PONG messages (yet)
		c.SetReadDeadline(time.Now().Add(b.readTimeout))

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
		default:
			log.WithFields(log.Fields{
				"message_type": msgType,
				"gateway_id":   gatewayID,
				"payload":      string(msg),
			}).Warning("backend/basicstation: unexpected message-type")
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

	g, err := b.gateways.get(gatewayID)
	if err != nil {
		log.WithError(err).WithField("gateway_id", gatewayID).Error("backend/basicstation: get gateway error")
		return
	}

	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		log.WithError(err).Error("backend/basicstation: get timestamp proto error")
		return
	}

	// TODO: remove this in the next major release
	if b.routerConfig == nil {
		b.gatewayStatsChan <- gw.GatewayStats{
			GatewayId:     gatewayID[:],
			Ip:            g.conn.RemoteAddr().String(),
			Time:          ts,
			ConfigVersion: g.configVersion,
		}

		return
	}

	websocketSendCounter("router_config").Inc()
	if err := b.sendToGateway(gatewayID, *b.routerConfig); err != nil {
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
	txack.DownlinkId = b.diidMap[uint16(v.DIID)]

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

func (b *Backend) sendToGateway(gatewayID lorawan.EUI64, v interface{}) error {
	gw, err := b.gateways.get(gatewayID)
	if err != nil {
		return errors.Wrap(err, "get gateway error")
	}

	gw.conn.SetWriteDeadline(time.Now().Add(b.writeTimeout))
	if err := gw.conn.WriteJSON(v); err != nil {
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
			}
		}
	}()

	handler(r, conn)
}
