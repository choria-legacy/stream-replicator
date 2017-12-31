package memory

import (
	"sync"
	"time"

	stan "github.com/nats-io/go-nats-streaming"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/tidwall/gjson"
	"github.com/sirupsen/logrus"
)

type Limiter struct {
	key  string
	age  time.Duration
	topic string
	seen map[string]time.Time
	mu   *sync.Mutex
}

var seenGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "stream_replicator_limiter_memory_seen",
	Help: "How many unique values were seen in the inspect key",
}, []string{"key", "name"})

var skippedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "stream_replicator_limiter_memory_skipped",
	Help: "How many times the limiter skipped a message that would have been published",
}, []string{"key", "name"})

var passedCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "stream_replicator_limiter_memory_passed",
	Help: "How many times the limiter passed a message for processing",
}, []string{"key", "name"})

var errCtr = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "stream_replicator_limiter_memory_errors",
	Help: "How many errors were encountered during processing messages",
}, []string{"key", "name"})

func init() {
	prometheus.MustRegister(seenGauge)
	prometheus.MustRegister(skippedCtr)
	prometheus.MustRegister(passedCtr)
	prometheus.MustRegister(errCtr)
}

func (m *Limiter) Configure(key string, age time.Duration, topic string) error {
	m.mu = &sync.Mutex{}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.key = key
	m.age = age
	m.topic = topic

	m.seen = make(map[string]time.Time)

	go m.scrub()
	go m.promUpdater()

	return nil
}

func (m *Limiter) ProcessAndRecord(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error {
	if m.key == "" {
		passedCtr.WithLabelValues(m.key, m.topic).Inc()
		return f(msg, true)
	}

	value := gjson.GetBytes(msg.Data, m.key).String()
	process := m.shouldProcess(value)

	if process {
		passedCtr.WithLabelValues(m.key, m.topic).Inc()
	} else {
		skippedCtr.WithLabelValues(m.key, m.topic).Inc()
	}

	err := f(msg, process)
	if err != nil {
		errCtr.WithLabelValues(m.key, m.topic).Inc()
		return err
	}

	if process {
		m.mu.Lock()
		m.seen[value] = time.Now()
		m.mu.Unlock()
	}

	return nil
}

func (m *Limiter) shouldProcess(value string) bool {
	if value == "" {
		return true
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	t, found := m.seen[value]
	if !found {
		return true
	}

	oldest := time.Now().Add(-1 * m.age)

	if t.Before(oldest) {
		return true
	}

	logrus.Debugf("Skipping message due to %s=%s last seen %s > %s", m.key, value, t, oldest)

	return false
}

func (m *Limiter) promUpdater() {
	ticker := time.NewTicker(10 * time.Second)

	for range ticker.C {
		m.mu.Lock()
		seenGauge.WithLabelValues(m.key, m.topic).Set(float64(len(m.seen)))
		m.mu.Unlock()
	}
}

func (m *Limiter) scrubber() {
	m.mu.Lock()
	defer m.mu.Unlock()

	killtime := time.Now().Add((-1 * m.age) - (10 * time.Minute))

	for i, t := range m.seen {
		if t.Before(killtime) {
			delete(m.seen, i)
		}
	}
}

func (m *Limiter) scrub() {
	ticker := time.NewTicker(1 * time.Minute)

	for range ticker.C {
		m.scrubber()
	}
}
