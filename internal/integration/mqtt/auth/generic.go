package auth

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	"github.com/brocaar/lora-gateway-bridge/internal/config"
)

// GenericAuthentication implements a generic MQTT authentication.
type GenericAuthentication struct {
	server       string
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
		tlsConfig: tlsConfig,

		server:       conf.Integration.MQTT.Auth.Generic.Server,
		username:     conf.Integration.MQTT.Auth.Generic.Username,
		password:     conf.Integration.MQTT.Auth.Generic.Password,
		cleanSession: conf.Integration.MQTT.Auth.Generic.CleanSession,
		clientID:     conf.Integration.MQTT.Auth.Generic.ClientID,
	}, nil
}

// Init applies the initial configuration.
func (a *GenericAuthentication) Init(opts *mqtt.ClientOptions) error {
	opts.AddBroker(a.server)
	opts.SetUsername(a.username)
	opts.SetPassword(a.password)
	opts.SetCleanSession(a.cleanSession)
	opts.SetClientID(a.clientID)

	if a.tlsConfig != nil {
		opts.SetTLSConfig(a.tlsConfig)
	}

	return nil
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
