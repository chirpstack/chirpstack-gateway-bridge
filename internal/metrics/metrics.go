package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/chirpstack-gateway-bridge/internal/config"
)

// Setup configures the metrics package.
func Setup(conf config.Config) error {
	if !conf.Metrics.Prometheus.EndpointEnabled {
		return nil
	}

	log.WithFields(log.Fields{
		"bind": conf.Metrics.Prometheus.Bind,
	}).Info("metrics: starting prometheus metrics server")

	server := http.Server{
		Handler: promhttp.Handler(),
		Addr:    conf.Metrics.Prometheus.Bind,
	}

	go func() {
		err := server.ListenAndServe()
		log.WithError(err).Error("metrics: prometheus metrics server error")
	}()

	return nil
}
