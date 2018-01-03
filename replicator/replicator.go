package replicator

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/limiter"
	"github.com/choria-io/stream-replicator/limiter/memory"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Copier is a single instance of a topic replicator
type Copier struct {
	config config.TopicConf
	tls    bool
	Log    *logrus.Entry
	ctx    context.Context
	cancel func()
}

// Setup validates the configuration of the copier and sets defaults where possible
func (c *Copier) Setup(name string, topic config.TopicConf) error {
	c.config = topic
	c.tls = config.TLS()

	if c.config.Topic == "" {
		return fmt.Errorf("A topic is required")
	}

	if c.config.SourceURL == "" {
		c.config.SourceURL = "nats://localhost:4222"
	}

	if c.config.SourceID == "" {
		return fmt.Errorf("A from cluster id is required")
	}

	if c.config.TargetURL == "" {
		return fmt.Errorf("A destination URL is required")
	}

	if c.config.TargetID == "" {
		return fmt.Errorf("A target cluster id is required")
	}

	if c.config.Workers == 0 {
		c.config.Workers = 1
	}

	if c.config.Name == "" {
		c.config.Name = fmt.Sprintf("%s_%s_stream_replicator", name, strings.Replace(c.config.Topic, ".", "_", -1))
	}

	if c.config.Workers > 1 {
		c.config.Queued = true
	}

	if c.config.Queued && c.config.QueueGroup == "" {
		c.config.QueueGroup = fmt.Sprintf("%s_stream_replicator_grp", strings.Replace(c.config.Topic, ".", "_", -1))
	}

	c.Log = logrus.WithFields(logrus.Fields{"topic": c.config.Topic, "workers": c.config.Workers, "name": c.config.Name, "queue": c.config.QueueGroup})

	c.ctx, c.cancel = context.WithCancel(context.Background())

	if c.config.Inspect != "" && c.config.MinAge != "" {
		c.Log.Infof("Configuring limiter with on key %s with min age %s", c.config.Inspect, c.config.MinAge)
		limiter.Configure(c.ctx, c.config, &memory.Limiter{})
	}

	return nil
}

// Run starts all the worker in a replicator
func (c *Copier) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < c.config.Workers; i++ {
		w := newWorker(i, c.config, c.tls, c.Log)
		wg.Add(1)
		go w.Run(ctx, wg)
	}

	select {
	case <-ctx.Done():
		c.cancel()
	}
}

// SetupPrometheus starts a prometheus exporter
func (c *Copier) SetupPrometheus(port int) {
	c.Log.Infof("Listening for /metrics on %d", port)
	http.Handle("/metrics", promhttp.Handler())
	c.Log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
