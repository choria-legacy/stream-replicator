package connector

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	streamReconnectCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_stream_reconnections",
		Help: "Number of times the NATS Stream reconnected",
	}, []string{"name", "worker"})

	reconnectCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_connection_reconnections",
		Help: "Number of times the connector reconnected to the middleware",
	}, []string{"name", "worker"})

	closedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_connection_closed",
		Help: "Number of times the connection was closed",
	}, []string{"name", "worker"})

	errorCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_connection_errors",
		Help: "Number of times the connection encountered an error",
	}, []string{"name", "worker"})
)

func init() {
	prometheus.MustRegister(reconnectCtr)
	prometheus.MustRegister(closedCtr)
	prometheus.MustRegister(errorCtr)
	prometheus.MustRegister(streamReconnectCtr)
}
