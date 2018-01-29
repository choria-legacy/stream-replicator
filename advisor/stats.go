package advisor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	timeoutAdvisoryCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_timeout",
		Help: "Number of times we notified about uncommunicative nodes",
	}, []string{"name"})

	recoverAdvisoryCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_recover",
		Help: "Number of times we notified about nodes that recovered before the expiry age",
	}, []string{"name"})

	expiredAdvisoryCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_expire",
		Help: "Number of times a node did not recover before its expiry deadline",
	}, []string{"name"})

	publishErrCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_errors",
		Help: "Number of times publishing an advisory failed",
	}, []string{"name"})
)

func init() {
	prometheus.MustRegister(timeoutAdvisoryCtr)
	prometheus.MustRegister(recoverAdvisoryCtr)
	prometheus.MustRegister(expiredAdvisoryCtr)
	prometheus.MustRegister(publishErrCtr)
}
