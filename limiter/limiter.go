package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/stream-replicator/config"
	stan "github.com/nats-io/go-nats-streaming"
)

var inspecter Inspecter

var key string
var age time.Duration
var topic string

type Inspecter interface {
	Configure(ctx context.Context, wg *sync.WaitGroup, inspectKey string, updateFlagKey string, age time.Duration, topic string) error
	ProcessAndRecord(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error
}

// Configure configures a given limiter
func Configure(ctx context.Context, wg *sync.WaitGroup, c *config.TopicConf, ins Inspecter) error {
	d, err := time.ParseDuration(c.MinAge)
	if err != nil {
		return fmt.Errorf("could not parse duration '%s': %s", c.MinAge, err)
	}

	err = ins.Configure(ctx, wg, c.Inspect, c.UpdateFlag, d, c.Name)
	if err != nil {
		return fmt.Errorf("could not configure inspecter: %s", err)
	}

	inspecter = ins

	return nil
}

func Process(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error {
	if inspecter == nil {
		return f(msg, true)
	}

	return inspecter.ProcessAndRecord(msg, f)
}
