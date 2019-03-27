package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

type AzureIoTHubConfig struct {
	DeviceID      string `mapstructure:"device_id"`
	IOTHubname    string `mapstructure:"iot_hub_name"`
	CACert        string `mapstructure:"iot_hub_ca_file"`
	DeviceKeyFile string `mapstructure:"device_key_file"`
}

type AzureIoTHubAuthentication struct {
	clientID  string
	username  string
	deviceKey []byte
	tlsConfig *tls.Config
	config    AzureIoTHubConfig
}

func createSASToken(uri string, saKey []byte) string {
	encoded := url.QueryEscape(uri)
	now := time.Now().Unix()
	// 24 hours token
	day := 60 * 60 * 24
	ts := now + int64(day)
	expiry := strconv.Itoa(int(ts))
	signature := encoded + "\n" + expiry
	b64key, err := b64.StdEncoding.DecodeString(string(saKey))
	if err != nil {
		log.Panicf("Azure IoT Key cannot be decoded")
	}

	mac := hmac.New(sha256.New, b64key)
	mac.Write([]byte(signature))
	hash := url.QueryEscape(b64.StdEncoding.EncodeToString(mac.Sum(nil)))

	// IoT Hub SAS Token only needs `sr`, `sig` and `se` unlike other Azure servies
	token := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%s",
		encoded,
		hash,
		expiry,
	)

	return token
}

func NewAzureIoTHubAuthentication(config AzureIoTHubConfig) (Authentication, error) {
	tlsConfig, err := newTLSConfig(config.CACert, "", "")
	if err != nil {
		return nil, errors.Wrap(err, "read ca cert")
	}

	deviceKey, err := ioutil.ReadFile(config.DeviceKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading device key file")
	}

	username := fmt.Sprintf("%s.azure-devices.net/%s",
		config.IOTHubname,
		config.DeviceID,
	)

	return &AzureIoTHubAuthentication{
		clientID:  config.DeviceID,
		username:  username,
		deviceKey: deviceKey,
		tlsConfig: tlsConfig,
		config:    config,
	}, nil
}

func (a *AzureIoTHubAuthentication) Init(opts *mqtt.ClientOptions) error {
	broker := fmt.Sprintf("ssl://%s.azure-devices.net:8883", a.config.IOTHubname)
	opts.AddBroker(broker)
	opts.SetClientID(a.clientID)
	opts.SetUsername(a.username)

	return nil
}

func (a *AzureIoTHubAuthentication) Update(opts *mqtt.ClientOptions) error {
	resourceURI := fmt.Sprintf("%s.azure-devices.net/devices/%s",
		a.config.IOTHubname,
		a.config.DeviceID,
	)
	token := createSASToken(resourceURI, a.deviceKey)
	log.Info("azure: updating")
	opts.SetPassword(token)

	return nil
}

func (a *AzureIoTHubAuthentication) ReconnectAfter() time.Duration {
	return 24 * time.Hour
}
