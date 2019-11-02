package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
)

// See:
// https://docs.microsoft.com/en-us/azure/iot-hub/iot-hub-mqtt-support#tlsssl-configuration
// https://github.com/Azure/azure-iot-sdk-c/blob/master/certs/certs.c
const digiCertBaltimoreRootCA = `
-----BEGIN CERTIFICATE-----
MIIDdzCCAl+gAwIBAgIEAgAAuTANBgkqhkiG9w0BAQUFADBaMQswCQYDVQQGEwJJ
RTESMBAGA1UEChMJQmFsdGltb3JlMRMwEQYDVQQLEwpDeWJlclRydXN0MSIwIAYD
VQQDExlCYWx0aW1vcmUgQ3liZXJUcnVzdCBSb290MB4XDTAwMDUxMjE4NDYwMFoX
DTI1MDUxMjIzNTkwMFowWjELMAkGA1UEBhMCSUUxEjAQBgNVBAoTCUJhbHRpbW9y
ZTETMBEGA1UECxMKQ3liZXJUcnVzdDEiMCAGA1UEAxMZQmFsdGltb3JlIEN5YmVy
VHJ1c3QgUm9vdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKMEuyKr
mD1X6CZymrV51Cni4eiVgLGw41uOKymaZN+hXe2wCQVt2yguzmKiYv60iNoS6zjr
IZ3AQSsBUnuId9Mcj8e6uYi1agnnc+gRQKfRzMpijS3ljwumUNKoUMMo6vWrJYeK
mpYcqWe4PwzV9/lSEy/CG9VwcPCPwBLKBsua4dnKM3p31vjsufFoREJIE9LAwqSu
XmD+tqYF/LTdB1kC1FkYmGP1pWPgkAx9XbIGevOF6uvUA65ehD5f/xXtabz5OTZy
dc93Uk3zyZAsuT3lySNTPx8kmCFcB5kpvcY67Oduhjprl3RjM71oGDHweI12v/ye
jl0qhqdNkNwnGjkCAwEAAaNFMEMwHQYDVR0OBBYEFOWdWTCCR1jMrPoIVDaGezq1
BE3wMBIGA1UdEwEB/wQIMAYBAf8CAQMwDgYDVR0PAQH/BAQDAgEGMA0GCSqGSIb3
DQEBBQUAA4IBAQCFDF2O5G9RaEIFoN27TyclhAO992T9Ldcw46QQF+vaKSm2eT92
9hkTI7gQCvlYpNRhcL0EYWoSihfVCr3FvDB81ukMJY2GQE/szKN+OMY3EU/t3Wgx
jkzSswF07r51XgdIGn9w/xZchMB5hbgF/X++ZRGjD8ACtPhSNzkE1akxehi/oCr0
Epn3o0WC4zxe9Z2etciefC7IpJ5OCBRLbf1wbWsaY71k5h+3zvDyny67G7fyUIhz
ksLi4xaNmjICq44Y3ekQEe5+NauQrz4wlHrQMz2nZQ/1/I6eYs9HRCwBXbsdtTLS
R9I4LtD+gdwyah617jzV/OeBHRnDJELqYzmp
-----END CERTIFICATE-----
`

type authType int

const (
	authTypeSymmetric authType = iota
	authTypeX509
)

// AzureIoTHubAuthentication implements the Azure IoT Hub authentication.
type AzureIoTHubAuthentication struct {
	authType authType

	clientID           string
	username           string
	deviceKey          []byte
	hostname           string
	sasTokenExpiration time.Duration

	tlsConfig *tls.Config
}

// NewAzureIoTHubAuthentication creates an AzureIoTHubAuthentication.
func NewAzureIoTHubAuthentication(c config.Config) (Authentication, error) {
	var auth AzureIoTHubAuthentication

	at := authTypeSymmetric
	conf := c.Integration.MQTT.Auth.AzureIoTHub

	certpool := x509.NewCertPool()
	if !certpool.AppendCertsFromPEM([]byte(digiCertBaltimoreRootCA)) {
		return nil, errors.New("append ca cert from pem error")
	}
	tlsConfig := tls.Config{
		RootCAs: certpool,
	}

	if conf.TLSCert != "" || conf.TLSKey != "" {
		at = authTypeX509
	}

	if at == authTypeSymmetric {
		if conf.DeviceConnectionString != "" {
			kvMap, err := parseConnectionString(conf.DeviceConnectionString)
			if err != nil {
				return nil, errors.Wrap(err, "parse connection string error")
			}

			for k, v := range kvMap {
				switch k {
				case "HostName":
					conf.Hostname = v
				case "DeviceId":
					conf.DeviceID = v
				case "SharedAccessKey":
					conf.DeviceKey = v
				}
			}
		}

		deviceKeyB, err := base64.StdEncoding.DecodeString(conf.DeviceKey)
		if err != nil {
			return nil, errors.Wrap(err, "decode device key error")
		}

		auth.deviceKey = deviceKeyB
		auth.sasTokenExpiration = conf.SASTokenExpiration
	}

	if at == authTypeX509 {
		kp, err := tls.LoadX509KeyPair(conf.TLSCert, conf.TLSKey)
		if err != nil {
			return nil, errors.Wrap(err, "load tls key-pair error")
		}

		tlsConfig.Certificates = []tls.Certificate{kp}
	}

	auth.clientID = conf.DeviceID
	auth.hostname = conf.Hostname
	auth.tlsConfig = &tlsConfig
	auth.username = fmt.Sprintf("%s/%s", conf.Hostname, conf.DeviceID)

	return &auth, nil
}

// Init applies the initial configuration.
func (a *AzureIoTHubAuthentication) Init(opts *mqtt.ClientOptions) error {
	broker := fmt.Sprintf("ssl://%s:8883", a.hostname)
	opts.AddBroker(broker)
	opts.SetClientID(a.clientID)
	opts.SetUsername(a.username)
	opts.SetTLSConfig(a.tlsConfig)

	return nil
}

// Update updates the authentication options.
func (a *AzureIoTHubAuthentication) Update(opts *mqtt.ClientOptions) error {
	if a.authType == authTypeSymmetric {
		resourceURI := fmt.Sprintf("%s/devices/%s",
			a.hostname,
			a.clientID,
		)
		token, err := createSASToken(resourceURI, a.deviceKey, a.sasTokenExpiration)
		if err != nil {
			return errors.Wrap(err, "create SAS token error")
		}

		opts.SetPassword(token)
	}

	return nil
}

// ReconnectAfter returns a time.Duration after which the MQTT client must re-connect.
// Note: return 0 to disable the periodical re-connect feature.
func (a *AzureIoTHubAuthentication) ReconnectAfter() time.Duration {
	return a.sasTokenExpiration
}

func createSASToken(uri string, deviceKey []byte, expiration time.Duration) (string, error) {
	encoded := url.QueryEscape(uri)
	exp := time.Now().Add(expiration).Unix()

	signature := fmt.Sprintf("%s\n%d", encoded, exp)

	mac := hmac.New(sha256.New, deviceKey)
	mac.Write([]byte(signature))
	hash := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	// IoT Hub SAS Token only needs `sr`, `sig` and `se` unlike other Azure services
	token := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%d",
		encoded,
		hash,
		exp,
	)

	return token, nil
}

func parseConnectionString(str string) (map[string]string, error) {
	out := make(map[string]string)
	pairs := strings.Split(str, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("expected two items in: %+v", kv)
		}

		out[kv[0]] = kv[1]
	}

	return out, nil
}
