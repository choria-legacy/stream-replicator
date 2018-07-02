package advisor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/stream-replicator/backoff"
	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/connector"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
)

// AgeAdvisoryV1 defines a message published when a node has not been seen within configured deadlines and when it recovers
type AgeAdvisoryV1 struct {
	Version    string    `json:"$schema"`
	Inspect    string    `json:"inspect"`
	Value      string    `json:"value"`
	Age        int64     `json:"age"`
	Seen       int64     `json:"seen"`
	Replicator string    `json:"replicator"`
	Timestamp  int64     `json:"timestamp"`
	Event      EventType `json:"event" validate:"enum=timeout,recover,expire"`
}

// EventType is the kind of event that triggered the advisory
type EventType string

const (
	// Timeout is the event that happens once a node has not been seen
	// for longer than the topic maximum age
	Timeout = EventType("timeout")

	// Recovery is the event that happens if a node has previously been
	// advised about and it came back before Expiry happens
	Recovery = EventType("recover")

	// Expired is the event that happens if a node has not been seen for
	// longer than the max age on the topic
	Expired = EventType("expire")
)

var out chan AgeAdvisoryV1

var seen map[string]time.Time
var advised map[string]time.Time

var mu = &sync.Mutex{}
var configured = false
var conf *config.TopicConf
var interval time.Duration
var age time.Duration
var err error
var log *logrus.Entry
var conn stan.Conn
var natstls bool
var name string

func init() {
	reset()
}

// Configure configures the advisor
func Configure(tls bool, c *config.TopicConf) error {
	mu.Lock()
	defer mu.Unlock()

	conf = c
	name = fmt.Sprintf("%s_advisor", c.Name)
	log = logrus.WithFields(logrus.Fields{"name": name})

	if conf.Advisory == nil {
		log.Warn("No advisory settings configured, disabling advisory publishing")
		return nil
	}

	natstls = tls

	age, err = time.ParseDuration(c.Advisory.Age)
	if err != nil {
		return fmt.Errorf("age cannot be parsed as a duration: %s", err)
	}

	interval, err = time.ParseDuration(c.MinAge)
	if err != nil {
		return fmt.Errorf("topic min age cannot be parsed as a duration: %s", err)
	}

	reset()

	configured = true

	return nil
}

// Connect initiates the connection to NATS Streaming
func Connect(ctx context.Context, wg *sync.WaitGroup) {
	mu.Lock()
	defer mu.Unlock()

	if !configured {
		return
	}

	log.Debug("Starting advisor connection")
	connect(ctx)

	log.Debug("Starting advisor publisher")
	wg.Add(1)
	go publisher(ctx, wg)

	log.Debug("Starting advisor monitor")
	wg.Add(1)
	go monitor(ctx, wg)
}

// Record records the fact that a node was seen
func Record(id string) {
	RecordTime(id, time.Now())
}

// RecordTime records that a sender was seen at a specific time
func RecordTime(id string, seent time.Time) {
	mu.Lock()
	defer mu.Unlock()

	if !configured {
		return
	}

	// the limiters will have thresholds and grace periods etc,
	// we have none so if they send us old stuff just drop it
	if seent.Before(oldest()) {
		return
	}

	// we previously advised about this node, so
	// its back now lets advise about it and delete
	// the advisory record
	t, ok := advised[id]
	if ok {
		log.Infof("sending advisory: %s: returned after previous advisory at %v", id, t)
		recoverAdvisoryCtr.WithLabelValues(name).Inc()

		out <- newAdvisory(id, Recovery)
		delete(advised, id)
	}

	log.Debugf("Recorded %s as seen at %v", id, seent)

	seen[id] = seent
}

// once a minute goes runs the adviser
func monitor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if !configured {
		return
	}

	log.Debug("Starting advisor monitor")

	ticker := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ticker.C:
			log.Debug("Starting advisory loop")
			advise()
		case <-ctx.Done():
			return
		}
	}
}

// goes through all the nodes in the seen list, find the ones
// last seen > the advisery trigger time sends an advisory for them
func advise() {
	mu.Lock()
	defer mu.Unlock()

	if !configured {
		return
	}

	timeout := time.Now().Add(0 - age)
	expire := time.Now().Add(0 - interval)

	log.Debug("Looking for nodes last seen earlier than %v", timeout)

	for i, t := range seen {
		if t.Before(expire) {
			log.Infof("sending advisory: %s: expiring", i)
			expiredAdvisoryCtr.WithLabelValues(name).Inc()

			out <- newAdvisory(i, Expired)

			delete(seen, i)
			delete(advised, i)

			continue
		}

		if t.Before(timeout) {
			_, found := advised[i]

			if !found {
				advisory := newAdvisory(i, Timeout)

				log.Infof("sending advisory: %s: older than %v, last seen %d seconds ago", i, timeout, advisory.Age)
				timeoutAdvisoryCtr.WithLabelValues(name).Inc()

				out <- advisory
				advised[i] = time.Now()
			}
		}
	}
}

func newAdvisory(id string, event EventType) AgeAdvisoryV1 {
	return AgeAdvisoryV1{
		Timestamp:  time.Now().UTC().Unix(),
		Age:        time.Now().Unix() - seen[id].Unix(),
		Inspect:    conf.Inspect,
		Replicator: conf.Name,
		Seen:       seen[id].Unix(),
		Value:      id,
		Event:      event,
		Version:    "https://choria.io/schemas/sr/v1/age_advisory.json",
	}
}

func connect(ctx context.Context) {
	var c *connector.Connection

	if conf.Advisory.Cluster == "source" {
		log.Infof("Connection to source to publish advisories")
		c = connector.New(name, natstls, connector.Source, conf, log)
	} else {
		log.Infof("Connection to target to publish advisories")
		c = connector.New(name, natstls, connector.Target, conf, log)
	}

	conn = c.Connect(ctx)
}

func publisher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case msg := <-out:
			d, err := json.Marshal(msg)
			if err != nil {
				log.Errorf("Cannot publish advisory: %s", err)
				publishErrCtr.WithLabelValues(name).Inc()
				continue
			}

			for i := 0; i < 10; i++ {
				err := conn.Publish(conf.Advisory.Target, d)
				if err != nil {
					log.Warnf("Failed to publish %s advisory for %s: %s", msg.Event, msg.Value, err)
					publishErrCtr.WithLabelValues(name).Inc()

					if i < 9 {
						if backoff.FiveSec.InterruptableSleep(ctx, i) != nil {
							break
						}
					}

					continue
				}

				break
			}

		case <-ctx.Done():
			log.Infof("Advisor shutting down")
			conn.Close()
			return
		}
	}
}

func reset() {
	out = make(chan AgeAdvisoryV1, 1000)
	seen = make(map[string]time.Time)
	advised = make(map[string]time.Time)
	configured = false
}

func oldest() time.Time {
	return time.Now().Add(0 - age)
}
