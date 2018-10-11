package auth

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// GenericConfig defines the generic configuration.
type GenericConfig struct {
	Server               string
	Username             string
	Password             string
	CACert               string        `mapstructure:"ca_cert"`
	TLSCert              string        `mapstructure:"tls_cert"`
	TLSKey               string        `mapstructure:"tls_key"`
	QOS                  uint8         `mapstructure:"qos"`
	CleanSession         bool          `mapstructure:"clean_session"`
	ClientID             string        `mapstructure:"client_id"`
	MaxReconnectInterval time.Duration `mapstructure:"max_reconnect_interval"`
}

// GenericAuthentication implements a generic MQTT authentication.
type GenericAuthentication struct {
	config    GenericConfig
	tlsConfig *tls.Config
}

// NewGenericAuthentication creates a GenericAuthentication.
func NewGenericAuthentication(config GenericConfig) (Authentication, error) {
	tlsConfig, err := newTLSConfig(config.CACert, config.TLSCert, config.TLSKey)
	if err != nil {
		return nil, errors.Wrap(err, "mqtt/auth: new tls config error")
	}

	return &GenericAuthentication{
		config:    config,
		tlsConfig: tlsConfig,
	}, nil
}

// Init applies the initial configuration.
func (a *GenericAuthentication) Init(opts *mqtt.ClientOptions) error {
	opts.AddBroker(a.config.Server)
	opts.SetUsername(a.config.Username)
	opts.SetPassword(a.config.Password)
	opts.SetCleanSession(a.config.CleanSession)
	opts.SetClientID(a.config.ClientID)
	opts.SetMaxReconnectInterval(a.config.MaxReconnectInterval)

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
