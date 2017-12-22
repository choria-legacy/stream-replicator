package replicator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/choria-io/stream-replicator/backoff"
	nats "github.com/nats-io/go-nats"
	stan "github.com/nats-io/go-nats-streaming"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type worker struct {
	name string

	from   stan.Conn
	to     stan.Conn
	copier *Copier
	log    *logrus.Entry
	sub    stan.Subscription
}

func NewWorker(i int, c *Copier) *worker {
	w := worker{
		name:   fmt.Sprintf("%s_%d", c.Name, i),
		log:    c.Log.WithFields(logrus.Fields{"worker": i}),
		copier: c,
	}

	return &w
}

func (w *worker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	err := w.connect(ctx)
	if err != nil {
		w.log.Errorf("Could not start worker: %s", err.Error())
		return
	}

	err = w.subscribe()
	if err != nil {
		w.log.Errorf("Could not subscribe to source %s", w.copier.Topic)
		return
	}

	select {
	case <-ctx.Done():
		w.log.Infof("%s existing", w.name)
		w.sub.Unsubscribe()
		w.from.Close()
		w.to.Close()

		return
	}
}

func (w *worker) copyf(msg *stan.Msg) {
	obs := prometheus.NewTimer(processTime.WithLabelValues(w.name, w.copier.Name))
	defer obs.ObserveDuration()

	err := w.to.Publish(w.copier.Topic, msg.Data)
	if err != nil {
		w.log.Errorf("Could not publish message %d: %s", msg.Sequence, err.Error())
		failedCtr.WithLabelValues(w.name, w.copier.Name).Inc()
		return
	}

	copiedCtr.WithLabelValues(w.name, w.copier.Name).Inc()

	err = msg.Ack()
	if err != nil {
		ackFailedCtr.WithLabelValues(w.name, w.copier.Name).Inc()
		w.log.Errorf("Could not ack message %d: %s", msg.Sequence, err.Error())
	}
}

func (w *worker) subscribe() error {
	opts := []stan.SubscriptionOption{
		stan.DurableName(w.copier.Name),
		stan.DeliverAllAvailable(),
		stan.SetManualAckMode(),
	}

	var err error

	if w.copier.Queued {
		w.log.Infof("subscribing to %s in queue group %s", w.copier.Topic, w.copier.QueueGroup)
		w.sub, err = w.from.QueueSubscribe(w.copier.Topic, w.copier.QueueGroup, w.copyf, opts...)
	} else {
		w.log.Infof("subscribing to %s", w.copier.Topic)
		w.sub, err = w.from.Subscribe(w.copier.Topic, w.copyf, opts...)
	}

	return err
}

func (w *worker) connect(ctx context.Context) error {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		w.from = w.connectSTAN(ctx, w.copier.FromID, w.name, w.copier.From)
	}(wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		w.to = w.connectSTAN(ctx, w.copier.ToID, w.name, w.copier.To)
	}(wg)

	wg.Wait()

	if w.from == nil || w.to == nil {
		return fmt.Errorf("Could not establish initial connection to Stream brokers")
	}

	return nil
}

func (w *worker) connectSTAN(ctx context.Context, cid string, name string, urls string) stan.Conn {
	n := w.connectNATS(ctx, name, urls)
	if n == nil {
		w.log.Errorf("%s NATS connection could not be established, cannot connect to the Stream", name)
		return nil
	}

	var err error
	var conn stan.Conn
	try := 0

	for {
		try++

		conn, err = stan.Connect(cid, name, stan.NatsConn(n))
		if err != nil {
			w.log.Warnf("%s initial connection to the NATS Streaming broker cluster failed: %s", name, err.Error())

			if ctx.Err() != nil {
				w.log.Errorf("%s initial connection cancelled due to shut down", name)
				return nil
			}

			s := backoff.FiveSec.Duration(try)
			w.log.Infof("%s NATS Stream client sleeping %s after failed connection attempt %d", name, s, try)
			time.Sleep(s)

			continue
		}

		break
	}

	return conn
}

func (w *worker) connectNATS(ctx context.Context, name string, urls string) (natsc *nats.Conn) {
	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.Name(name),
		nats.DisconnectHandler(w.disconCb),
		nats.ReconnectHandler(w.reconCb),
		nats.ClosedHandler(w.closedCb),
		nats.ErrorHandler(w.errorCb),
	}

	var err error
	try := 0

	for {
		try++

		natsc, err = nats.Connect(urls, options...)
		if err != nil {
			w.log.Warnf("%s initial connection to the NATS broker cluster failed: %s", name, err.Error())

			if ctx.Err() != nil {
				w.log.Errorf("%s initial connection cancelled due to shut down", name)
				return nil
			}

			s := backoff.FiveSec.Duration(try)
			w.log.Infof("%s NATS client sleeping %s after failed connection attempt %d", name, s, try)
			time.Sleep(s)

			continue
		}

		w.log.Infof("%s NATS client connected to %s", name, natsc.ConnectedUrl())

		break
	}

	return
}

func (w *worker) disconCb(nc *nats.Conn) {
	err := nc.LastError()

	if err != nil {
		w.log.Warnf("%s NATS client connection got disconnected: %s", nc.Opts.Name, err.Error())
	} else {
		w.log.Warnf("%s NATS client connection got disconnected", nc.Opts.Name)
	}
}

func (w *worker) reconCb(nc *nats.Conn) {
	w.log.Warnf("%s NATS client reconnected after a previous disconnection, connected to %s", nc.Opts.Name, nc.ConnectedUrl())
	reconnectCtr.WithLabelValues(w.name, w.copier.Name).Inc()
}

func (w *worker) closedCb(nc *nats.Conn) {
	err := nc.LastError()

	if err != nil {
		w.log.Warnf("%s NATS client connection closed: %s", nc.Opts.Name, err.Error())
	} else {
		w.log.Warnf("%s NATS client connection closed", nc.Opts.Name)
	}

	closedCtr.WithLabelValues(w.name, w.copier.Name).Inc()
}

func (w *worker) errorCb(nc *nats.Conn, sub *nats.Subscription, err error) {
	w.log.Errorf("%s NATS client on %s encountered an error: %s", nc.Opts.Name, nc.ConnectedUrl(), err.Error())
	errorCtr.WithLabelValues(w.name, w.copier.Name).Inc()
}
