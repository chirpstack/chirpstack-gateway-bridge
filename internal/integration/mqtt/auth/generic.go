package auth

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
)

// GenericAuthentication implements a generic MQTT authentication.
type GenericAuthentication struct {
	servers      []string
	username     string
	password     string
	cleanSession bool
	clientID     string

	tlsConfig *tls.Config
}

// NewGenericAuthentication creates a GenericAuthentication.
func NewGenericAuthentication(conf config.Config) (Authentication, error) {
	tlsConfig, err := newTLSConfig(
		conf.Integration.MQTT.Auth.Generic.CACert,
		conf.Integration.MQTT.Auth.Generic.TLSCert,
		conf.Integration.MQTT.Auth.Generic.TLSKey,
	)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt/auth: new tls config error")
	}

	return &GenericAuthentication{
		tlsConfig:    tlsConfig,
		servers:      conf.Integration.MQTT.Auth.Generic.Servers,
		username:     conf.Integration.MQTT.Auth.Generic.Username,
		password:     conf.Integration.MQTT.Auth.Generic.Password,
		cleanSession: conf.Integration.MQTT.Auth.Generic.CleanSession,
		clientID:     conf.Integration.MQTT.Auth.Generic.ClientID,
	}, nil
}

// Init applies the initial configuration.
func (a *GenericAuthentication) Init(opts *mqtt.ClientOptions) error {
	for _, server := range a.servers {
		opts.AddBroker(server)
	}
	opts.SetUsername(a.username)
	opts.SetPassword(a.password)
	opts.SetCleanSession(a.cleanSession)
	opts.SetClientID(a.clientID)

	if a.tlsConfig != nil {
		opts.SetTLSConfig(a.tlsConfig)
	}

	return nil
}

// GetGatewayID returns the GatewayID if available.
func (a *GenericAuthentication) GetGatewayID() *lorawan.EUI64 {
	if a.clientID == "" {
		return nil
	}

	// Try to decode the client ID as gateway ID.
	var gatewayID lorawan.EUI64
	if err := gatewayID.UnmarshalText([]byte(a.clientID)); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"client_id": a.clientID,
		}).Warning("integration/mqtt/auth: could not decode client ID to gateway ID")
		return nil
	}

	return &gatewayID
}

// Update updates the authentication options.
func (a *GenericAuthentication) Update(opts *mqtt.ClientOptions) error {
	return nil
}

// ReconnectAfter returns a time.Duration after which the MQTT client must re-connect.
// Note: return 0 to disable the periodical re-connect feature.
func (a *GenericAuthentication) ReconnectAfter() time.Duration {
	return 0
}
