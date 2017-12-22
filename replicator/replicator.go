package replicator

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type Copier struct {
	Name       string
	Topic      string
	From       string
	FromID     string
	To         string
	ToID       string
	Workers    int
	Queued     bool
	QueueGroup string

	Log *logrus.Entry
}

// Setup validates the configuration of the copier and sets defaults where possible
func (c *Copier) Setup() error {
	if c.Topic == "" {
		return fmt.Errorf("A topic is required")
	}

	if c.From == "" {
		c.From = "nats://localhost:4222"
	}

	if c.FromID == "" {
		return fmt.Errorf("A from cluster id is required")
	}

	if c.To == "" {
		return fmt.Errorf("A destination URL is required")
	}

	if c.ToID == "" {
		return fmt.Errorf("A target cluster id is required")
	}

	if c.Workers == 0 {
		c.Workers = 1
	}

	if c.Name == "" {
		c.Name = fmt.Sprintf("%s_stream_replicator", strings.Replace(c.Topic, ".", "_", -1))
	}

	if c.Workers > 1 {
		c.Queued = true
	}

	if c.Queued && c.QueueGroup == "" {
		c.QueueGroup = fmt.Sprintf("%s_stream_replicator_grp", strings.Replace(c.Topic, ".", "_", -1))
	}

	c.Log = logrus.WithFields(logrus.Fields{"topic": c.Topic, "workers": c.Workers, "name": c.Name, "queue": c.QueueGroup})

	return nil
}

func (c *Copier) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for i := 0; i < c.Workers; i++ {
		w := NewWorker(i, c)
		wg.Add(1)
		go w.Run(ctx, wg)
	}
}
