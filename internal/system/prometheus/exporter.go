package prometheus

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func StartExporter(path string, port int) {
	log.WithFields(log.Fields{
		"path": path,
		"port": port,
	}).Info("starting prometheus stats exporter")

	http.Handle(path, promhttp.Handler())

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.WithFields(log.Fields{
		"path": path,
		"port": port,
	}).Errorf("failed to listen the prometheus exporter: %s", err)
}
