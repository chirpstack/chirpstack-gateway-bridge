package main

import (
	"time"

	"github.com/brocaar/chirpstack-gateway-bridge/cmd/chirpstack-gateway-bridge/cmd"
	paho "github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

type pahoLogWrapper struct {
	ln func(...interface{})
	f  func(string, ...interface{})
}

func (d pahoLogWrapper) Println(v ...interface{}) {
	d.ln(v...)
}

func (d pahoLogWrapper) Printf(format string, v ...interface{}) {
	d.f(format, v...)
}

func enableClientLogging() {
	l := log.WithField("module", "mqtt")
	paho.DEBUG = pahoLogWrapper{l.Debugln, l.Debugf}
	paho.ERROR = pahoLogWrapper{l.Errorln, l.Errorf}
	paho.WARN = pahoLogWrapper{l.Warningln, l.Warningf}
	paho.CRITICAL = pahoLogWrapper{l.Errorln, l.Errorf}
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	enableClientLogging()
}

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
