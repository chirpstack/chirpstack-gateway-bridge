package auth

import (
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// GCPCloudIoTCoreConfig devices the Cloud IoT Core configuration.
type GCPCloudIoTCoreConfig struct {
	Server        string
	DeviceID      string        `mapstructure:"device_id"`
	ProjectID     string        `mapstructure:"project_id"`
	CloudRegion   string        `mapstructure:"cloud_region"`
	RegistryID    string        `mapstructure:"registry_id"`
	JWTExpiration time.Duration `mapstructure:"jwt_expiration"`
	JWTKeyFile    string        `mapstructure:"jwt_key_file"`
}

// GCPCloudIoTCoreAuthentication implements the Google Cloud IoT Core authentication.
type GCPCloudIoTCoreAuthentication struct {
	siginingMethod *jwt.SigningMethodRSA
	privateKey     *rsa.PrivateKey
	clientID       string
	config         GCPCloudIoTCoreConfig
}

// NewGCPCloudIoTCoreAuthentication create a GCPCloudIoTCoreAuthentication.
func NewGCPCloudIoTCoreAuthentication(config GCPCloudIoTCoreConfig) (Authentication, error) {
	keyFileRaw, err := ioutil.ReadFile(config.JWTKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "read jwt key-file error")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyFileRaw)
	if err != nil {
		return nil, errors.Wrap(err, "parse jwt key-file error")
	}

	clientID := fmt.Sprintf("projects/%s/locations/%s/registries/%s/devices/%s",
		config.ProjectID,
		config.CloudRegion,
		config.RegistryID,
		config.DeviceID,
	)

	return &GCPCloudIoTCoreAuthentication{
		siginingMethod: jwt.SigningMethodRS256,
		privateKey:     privateKey,
		clientID:       clientID,
		config:         config,
	}, nil
}

// Init applies the initial configuration.
func (a *GCPCloudIoTCoreAuthentication) Init(opts *mqtt.ClientOptions) error {
	opts.AddBroker(a.config.Server)
	opts.SetClientID(a.clientID)
	return nil
}

// Update updates the authentication options.
func (a *GCPCloudIoTCoreAuthentication) Update(opts *mqtt.ClientOptions) error {
	token := jwt.NewWithClaims(a.siginingMethod, jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(a.ReconnectAfter()).Unix(),
		Audience:  a.config.ProjectID,
	})

	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		return errors.Wrap(err, "sign jwt token error")
	}

	opts.SetUsername(signedToken)
	opts.SetPassword(signedToken)

	return nil
}

// ReconnectAfter returns a time.Duration after which the MQTT client must re-connect.
// Note: return 0 to disable the periodical re-connect feature.
func (a *GCPCloudIoTCoreAuthentication) ReconnectAfter() time.Duration {
	return a.config.JWTExpiration
}
