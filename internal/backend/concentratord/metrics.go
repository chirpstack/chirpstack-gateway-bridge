package concentratord

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ec = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_concentratord_event_count",
		Help: "The number of received events (per type)",
	}, []string{"event"})

	cc = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "backend_concentratord_command_count",
		Help: "The number of received commands (per type)",
	}, []string{"command"})
)

func eventCounter(typ string) prometheus.Counter {
	return ec.With(prometheus.Labels{"event": typ})
}

func commandCounter(typ string) prometheus.Counter {
	return cc.With(prometheus.Labels{"command": typ})
}
