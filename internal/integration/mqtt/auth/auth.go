package auth

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	"github.com/brocaar/lorawan"
)

// Authentication defines the authentication interface.
type Authentication interface {
	// Init applies the initial configuration.
	Init(*mqtt.ClientOptions) error

	// GetGatewayID returns the GatewayID if available.
	GetGatewayID() *lorawan.EUI64

	// Update updates the authentication options.
	Update(*mqtt.ClientOptions) error

	// ReconnectAfter returns a time.Duration after which the MQTT client must re-connect.
	// Note: return 0 to disable the periodical re-connect feature.
	ReconnectAfter() time.Duration
}

func newTLSConfig(cafile, certFile, certKeyFile string) (*tls.Config, error) {
	if cafile == "" && certFile == "" && certKeyFile == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	if cafile != "" {
		cacert, err := ioutil.ReadFile(cafile)
		if err != nil {
			return nil, errors.Wrap(err, "load ca-cert error")
		}
		certpool := x509.NewCertPool()
		certpool.AppendCertsFromPEM(cacert)

		tlsConfig.RootCAs = certpool // RootCAs = certs used to verify server cert.
	}

	if certFile != "" && certKeyFile != "" {
		kp, err := tls.LoadX509KeyPair(certFile, certKeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "load tls key-pair error")
		}
		tlsConfig.Certificates = []tls.Certificate{kp}
	}

	return tlsConfig, nil
}
