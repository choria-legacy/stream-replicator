package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/choria-io/stream-replicator/advisor"

	"github.com/choria-io/stream-replicator/config"
	stan "github.com/nats-io/stan.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// Limiter is a in-process memory based state tracker that inspects
// data being processed, tracks a certain key and ensure a processor
// function is only run once per age per unique tracked key
//
// It can save the cache to disk regularly if configured and load
// it during startup which helps on very large sender counts to
// drastically reduce the restart costs of this kind of cache
type Limiter struct {
	key        string
	updateFlag string
	age        time.Duration
	topic      string
	statefile  string
	processed  map[string]time.Time
	mu         *sync.Mutex
	log        *logrus.Entry
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

func (m *Limiter) Configure(ctx context.Context, wg *sync.WaitGroup, inspectKey string, updateFlagKey string, age time.Duration, topic string) error {
	m.mu = &sync.Mutex{}
	m.updateFlag = updateFlagKey
	m.key = inspectKey
	m.age = age
	m.topic = topic
	m.log = logrus.WithFields(logrus.Fields{"key": inspectKey, "age": age, "topic": topic})

	if config.StateDirectory() != "" {
		m.statefile = filepath.Join(config.StateDirectory(), fmt.Sprintf("%s.json", topic))
	}

	m.processed = make(map[string]time.Time)

	m.readCache()

	wg.Add(1)
	go m.cacher(ctx, wg)
	wg.Add(1)
	go m.scrubber(ctx, wg)
	wg.Add(1)
	go m.promUpdater(ctx, wg)

	return nil
}

func (m *Limiter) ProcessAndRecord(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error {
	if m.key == "" {
		passedCtr.WithLabelValues(m.key, m.topic).Inc()
		return f(msg, true)
	}

	var process bool
	var value string

	res := gjson.GetBytes(msg.Data, m.updateFlag)
	if res.Exists() {
		process = res.Bool()
	}

	// even though we know we will update here should the updateFlag
	// be true we still need to figure out this value for advisory tracking
	res = gjson.GetBytes(msg.Data, m.key)
	if res.Exists() {
		value = res.String()
	}

	// but we do know the update flag is forcing the update so we should not
	// again decide based on the value of the key
	if !process {
		process = m.shouldProcess(value)
	}

	if process {
		passedCtr.WithLabelValues(m.key, m.topic).Inc()
	} else {
		skippedCtr.WithLabelValues(m.key, m.topic).Inc()
	}

	// we dont want to record advisories for data without the key
	// this might combine many different incorrect data items into
	// one bucket
	if value != "" {
		advisor.Record(value)
	}

	err := f(msg, process)
	if err != nil {
		errCtr.WithLabelValues(m.key, m.topic).Inc()
		return err
	}

	if process {
		m.mu.Lock()
		m.processed[value] = time.Now()
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

	t, found := m.processed[value]
	if !found {
		return true
	}

	oldest := time.Now().Add(0 - m.age)

	if t.Before(oldest) {
		return true
	}

	m.log.Debugf("Skipping message due to %s=%s last seen %s > %s", m.key, value, t, oldest)

	return false
}

func (m *Limiter) readCache() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.statefile == "" {
		m.log.Warn("No state_dir configured, last seen cache is not saved")
		return nil
	}

	if len(m.processed) > 0 {
		return fmt.Errorf("last seen cache is not empty")
	}

	d, err := ioutil.ReadFile(m.statefile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(d, &m.processed)
	if err != nil {
		return err
	}

	killtime := time.Now().Add(0 - 3*m.age)

	for i, t := range m.processed {
		if t.Before(killtime) {
			delete(m.processed, i)
			continue
		}

		advisor.RecordTime(i, t)
	}

	m.log.Infof("Read %d bytes of last-processed data from cache file %s.  After scrubbing old entries the last-processed data has %d entries.", len(d), m.statefile, len(m.processed))

	return nil
}

func (m *Limiter) writeCache() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.processed) == 0 {
		return nil
	}

	content, err := json.Marshal(m.processed)
	if err != nil {
		m.log.Errorf("Could not JSON encode last processed data: %s", err)
		return err
	}

	tmpfile, err := ioutil.TempFile(config.StateDirectory(), "memcache")
	if err != nil {
		m.log.Errorf("Could not create temp file: %s", err)
		return err
	}

	written, err := tmpfile.Write(content)
	if err != nil {
		m.log.Errorf("Could not write to temp file: %s", err)
		return err
	}

	err = tmpfile.Close()
	if err != nil {
		m.log.Errorf("Could not close temp file: %s", err)
		return err
	}

	m.log.Debugf("Wrote %d bytes to temp file %s", written, tmpfile.Name())

	err = os.Rename(tmpfile.Name(), m.statefile)
	if err != nil {
		m.log.Errorf("Could not rename file: %s", err)
		return err
	}

	m.log.Debugf("Wrote %d bytes to last seen cache %s", len(content), m.statefile)

	return nil
}

func (m *Limiter) cacher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if m.statefile == "" {
		m.log.Warn("Last seen timestamps cannot be saved, state_dir is not set")
		return
	}

	ticker := time.NewTicker(30 * time.Second)

	writer := func() {
		err := m.writeCache()
		if err != nil {
			m.log.Errorf("Could not write last seen data to cache: %s", err)
		}
	}

	for {
		select {
		case <-ticker.C:
			writer()

		case <-ctx.Done():
			m.log.Infof("Saving last seen state on exit")
			writer()

			return
		}
	}
}

func (m *Limiter) promUpdater(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			seenGauge.WithLabelValues(m.key, m.topic).Set(float64(len(m.processed)))
			m.mu.Unlock()

		case <-ctx.Done():
			return
		}
	}
}

func (m *Limiter) scrub() {
	m.mu.Lock()
	defer m.mu.Unlock()

	killtime := time.Now().Add(0 - 3*m.age)

	for i, t := range m.processed {
		if t.Before(killtime) {
			delete(m.processed, i)
		}
	}
}

func (m *Limiter) scrubber(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ticker.C:
			m.scrub()
		case <-ctx.Done():
			return
		}
	}
}
