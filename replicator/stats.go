package replicator

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	receivedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_received_msgs",
		Help: "How many messages were received",
	}, []string{"name", "worker"})

	receivedBytesCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_received_bytes",
		Help: "Size of messages that were received",
	}, []string{"name", "worker"})

	copiedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_copied_msgs",
		Help: "How many messages were copied",
	}, []string{"name", "worker"})

	copiedBytesCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_copied_bytes",
		Help: "Size of messages that were copied",
	}, []string{"name", "worker"})

	failedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_failed_msgs",
		Help: "How many messages failed to copy to the remote server",
	}, []string{"name", "worker"})

	ackFailedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_acks_failed",
		Help: "How many times ack'ing a message failed",
	}, []string{"name", "worker"})

	processTime = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "stream_replicator_processing_time",
		Help: "How long it took to process messages",
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

	sequenceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "stream_replicator_current_sequence",
		Help: "The current sequence number being copied",
	}, []string{"name", "worker"})
)

func init() {
	prometheus.MustRegister(receivedCtr)
	prometheus.MustRegister(receivedBytesCtr)
	prometheus.MustRegister(copiedCtr)
	prometheus.MustRegister(copiedBytesCtr)
	prometheus.MustRegister(failedCtr)
	prometheus.MustRegister(ackFailedCtr)
	prometheus.MustRegister(processTime)
	prometheus.MustRegister(reconnectCtr)
	prometheus.MustRegister(closedCtr)
	prometheus.MustRegister(errorCtr)
	prometheus.MustRegister(sequenceGauge)
}
