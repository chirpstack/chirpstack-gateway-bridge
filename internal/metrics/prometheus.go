package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// MustRegisterNewTimerWithError registers and returns a function for timing
// functions.
func MustRegisterNewTimerWithError(name, help string, labels []string) func(prometheus.Labels, func() error) error {
	labels = append(labels, "error")

	timer := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gw_" + name + "_duration_seconds",
		Help: help,
	}, labels)

	timer = prometheus.MustRegisterOrGet(timer).(*prometheus.HistogramVec)

	return func(labels prometheus.Labels, f func() error) error {
		labels["error"] = "false"
		start := time.Now()
		err := f()
		elasped := time.Since(start)

		if err != nil {
			labels["error"] = "true"
		}

		timer.With(labels).Observe(float64(elasped) / float64(time.Second))
		return err
	}
}

// MustRegisterNewCounter registers and returns a function for counting.
func MustRegisterNewCounter(name string, help string, labels []string) func(prometheus.Labels) {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gw_" + name + "_count",
		Help: help,
	}, labels)

	counter = prometheus.MustRegisterOrGet(counter).(*prometheus.CounterVec)

	return func(labels prometheus.Labels) {
		counter.With(labels).Inc()
	}
}
