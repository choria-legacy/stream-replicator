package limiter

import (
	"fmt"
	"time"

	"github.com/choria-io/stream-replicator/config"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
)

var inspecter Inspecter

var key string
var age time.Duration

type Inspecter interface {
	Configure(key string, age time.Duration) error
	ProcessAndRecord(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error
}

func Configure(c config.TopicConf, ins Inspecter) error {
	d, err := time.ParseDuration(c.MinAge)
	if err != nil {
		return fmt.Errorf("Could not parse duration '%s': %s", c.MinAge, err.Error())
	}

	err = ins.Configure(c.Inspect, d)
	if err != nil {
		return fmt.Errorf("Could not configure inspecter: %s", err.Error())
	}

	inspecter = ins

	return nil
}

func Process(msg *stan.Msg, f func(msg *stan.Msg, process bool) error) error {
	if inspecter == nil {
		logrus.Debug("Inspecter is not configured, running unconditionally")
		return f(msg, true)
	}

	return inspecter.ProcessAndRecord(msg, f)
}