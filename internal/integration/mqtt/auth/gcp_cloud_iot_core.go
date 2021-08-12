package auth

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
	"github.com/brocaar/lorawan"
)

// GCPCloudIoTCoreAuthentication implements the Google Cloud IoT Core authentication.
type GCPCloudIoTCoreAuthentication struct {
	siginingMethod *jwt.SigningMethodRSA
	privateKey     *rsa.PrivateKey
	clientID       string
	server         string
	projectID      string
	jwtExpiration  time.Duration
}

// NewGCPCloudIoTCoreAuthentication create a GCPCloudIoTCoreAuthentication.
func NewGCPCloudIoTCoreAuthentication(conf config.Config) (Authentication, error) {
	keyFileRaw, err := ioutil.ReadFile(conf.Integration.MQTT.Auth.GCPCloudIoTCore.JWTKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "read jwt key-file error")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyFileRaw)
	if err != nil {
		return nil, errors.Wrap(err, "parse jwt key-file error")
	}

	clientID := fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s",
		conf.Integration.MQTT.Auth.GCPCloudIoTCore.ProjectID,
		conf.Integration.MQTT.Auth.GCPCloudIoTCore.CloudRegion,
		conf.Integration.MQTT.Auth.GCPCloudIoTCore.RegistryID,
		conf.Integration.MQTT.Auth.GCPCloudIoTCore.DeviceID,
	)

	return &GCPCloudIoTCoreAuthentication{
		siginingMethod: jwt.SigningMethodRS256,
		privateKey:     privateKey,
		clientID:       clientID,
		server:         conf.Integration.MQTT.Auth.GCPCloudIoTCore.Server,
		projectID:      conf.Integration.MQTT.Auth.GCPCloudIoTCore.ProjectID,
		jwtExpiration:  conf.Integration.MQTT.Auth.GCPCloudIoTCore.JWTExpiration,
	}, nil
}

// Init applies the initial configuration.
func (a *GCPCloudIoTCoreAuthentication) Init(opts *mqtt.ClientOptions) error {
	opts.AddBroker(a.server)
	opts.SetClientID(a.clientID)
	return nil
}

// GetGatewayID returns the GatewayID if available.
// TODO: implement.
func (a *GCPCloudIoTCoreAuthentication) GetGatewayID() *lorawan.EUI64 {
	return nil
}

// Update updates the authentication options.
func (a *GCPCloudIoTCoreAuthentication) Update(opts *mqtt.ClientOptions) error {
	token := jwt.NewWithClaims(a.siginingMethod, jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(a.ReconnectAfter()).Unix(),
		Audience:  a.projectID,
	})

	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		return errors.Wrap(err, "sign jwt token error")
	}

	opts.SetUsername(signedToken)
	opts.SetPassword(signedToken)

	return nil
}

// ReconnectAfter returns a time.Duration after which the MQTT.Auth.client must re-connect.
// Note: return 0 to disable the periodical re-connect feature.
func (a *GCPCloudIoTCoreAuthentication) ReconnectAfter() time.Duration {
	return a.jwtExpiration
}
