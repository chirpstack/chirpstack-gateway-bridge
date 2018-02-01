package stats

import (
	"net/http"
	"github.com/fukata/golang-stats-api-handler"
	log "github.com/sirupsen/logrus"
	"fmt"
)

func StartMonitoringDrain(path string, port int) {
	log.WithFields(log.Fields{
		"path": path,
		"port": port,
	}).Info("starting system monitoring drain")

	http.HandleFunc(path, stats_api.Handler)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	log.WithFields(log.Fields{
		"path": path,
		"port": port,
	}).Errorf("failed to listen the system monitoring drain: %s", err)
}
