package replicator

import (
	"context"
	"fmt"
	"sync"

	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/connector"
	"github.com/choria-io/stream-replicator/limiter"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type worker struct {
	name string

	from   stan.Conn
	to     stan.Conn
	config config.TopicConf
	tls    bool
	log    *logrus.Entry
	sub    stan.Subscription
}

func newWorker(i int, config config.TopicConf, tls bool, log *logrus.Entry) *worker {
	w := worker{
		name:   fmt.Sprintf("%s_%d", config.Name, i),
		log:    log.WithFields(logrus.Fields{"worker": i}),
		config: config,
		tls:    tls,
	}

	return &w
}

func (w *worker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	err := w.connect(ctx)
	if err != nil {
		w.log.Errorf("Could not start worker: %s", err)
		return
	}

	err = w.subscribe()
	if err != nil {
		w.log.Errorf("Could not subscribe to source %s", w.config.Topic)
		return
	}

	select {
	case <-ctx.Done():
		w.log.Infof("%s existing", w.name)
		w.from.Close()
		w.to.Close()

		return
	}
}

func (w *worker) copyf(msg *stan.Msg) {
	obs := prometheus.NewTimer(processTime.WithLabelValues(w.name, w.config.Name))
	defer obs.ObserveDuration()

	receivedCtr.WithLabelValues(w.name, w.config.Name).Inc()
	receivedBytesCtr.WithLabelValues(w.name, w.config.Name).Add(float64(len(msg.Data)))

	limiter.Process(msg, func(msg *stan.Msg, process bool) error {
		if process {
			err := w.to.Publish(msg.Subject, msg.Data)
			if err != nil {
				w.log.Errorf("Could not publish message %d: %s", msg.Sequence, err)
				failedCtr.WithLabelValues(w.name, w.config.Name).Inc()
				return err
			}

			w.log.Debugf("Copied %d bytes in sequence %d from %s to %s", len(msg.Data), msg.Sequence, w.config.SourceURL, w.config.TargetURL)

			copiedBytesCtr.WithLabelValues(w.name, w.config.Name).Add(float64(len(msg.Data)))
			copiedCtr.WithLabelValues(w.name, w.config.Name).Inc()
		}

		sequenceGauge.WithLabelValues(w.name, w.config.Name).Set(float64(msg.Sequence))

		err := msg.Ack()
		if err != nil {
			ackFailedCtr.WithLabelValues(w.name, w.config.Name).Inc()
			w.log.Errorf("Could not ack message %d: %s", msg.Sequence, err)
		}

		return err
	})
}

func (w *worker) subscribe() error {
	opts := []stan.SubscriptionOption{
		stan.DurableName(w.config.Name),
		stan.DeliverAllAvailable(),
		stan.SetManualAckMode(),
		stan.MaxInflight(10),
	}

	var err error

	if w.config.Queued {
		w.log.Infof("subscribing to %s in queue group %s", w.config.Topic, w.config.QueueGroup)
		w.sub, err = w.from.QueueSubscribe(w.config.Topic, w.config.QueueGroup, w.copyf, opts...)
	} else {
		w.log.Infof("subscribing to %s", w.config.Topic)
		w.sub, err = w.from.Subscribe(w.config.Topic, w.copyf, opts...)
	}

	return err
}

func (w *worker) connect(ctx context.Context) error {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		c := connector.New(w.name, w.tls, connector.Source, w.config, w.log)
		w.from = c.Connect(ctx)
	}(wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		c := connector.New(w.name, w.tls, connector.Target, w.config, w.log)
		w.to = c.Connect(ctx)
	}(wg)

	wg.Wait()

	if w.from == nil || w.to == nil {
		return fmt.Errorf("Could not establish initial connection to Stream brokers")
	}

	return nil
}
