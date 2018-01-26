package advisor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	downAdvisoryCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_down",
		Help: "Number of times we notified about uncommunicative nodes",
	}, []string{"name"})

	upAdvisoryCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_up",
		Help: "Number of times we notified about nodes that recovered before the expiry age",
	}, []string{"name"})

	expiredAdvisoryCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stream_replicator_advisories_expired",
		Help: "Number of times a node did not recover before its expiry deadline",
	}, []string{"name"})
)

func init() {
	prometheus.MustRegister(downAdvisoryCtr)
	prometheus.MustRegister(upAdvisoryCtr)
	prometheus.MustRegister(expiredAdvisoryCtr)
}
