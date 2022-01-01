package connector

import (
	"context"
	"sync"
	"time"

	"github.com/choria-io/stream-replicator/backoff"
	"github.com/choria-io/stream-replicator/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/sirupsen/logrus"
)

// Connection holds a connection to NATS Streaming
type Connection struct {
	url  string
	log  *logrus.Entry
	conn stan.Conn
	name string
	cfg  *config.TopicConf
	id   string
	tls  bool
	subs []*subscription
	mu   *sync.Mutex
}

// Direction indicates which of the connectors to connect to
type Direction uint8

// Source indicates a connection to the source should be made
const Source = Direction(0)

// Target indicates a connection to the target should be made
const Target = Direction(1)

// New creates a new connector
func New(name string, tls bool, dir Direction, cfg *config.TopicConf, logger *logrus.Entry) *Connection {
	c := Connection{
		url:  cfg.TargetURL,
		log:  logger,
		name: name,
		id:   cfg.TargetID,
		tls:  tls,
		cfg:  cfg,
		subs: []*subscription{},
		mu:   &sync.Mutex{},
	}

	if dir == Source {
		c.url = cfg.SourceURL
		c.id = cfg.SourceID
	}

	return &c
}

// NatsConn returns the active nats connection
func (c *Connection) NatsConn() *nats.Conn {
	return c.conn.NatsConn()
}

// Connect connects to the configured stream
func (c *Connection) Connect(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connectSTAN(ctx)
}

// Subscribe subscribes to a subject, if group is empty a normal subscription is done
func (c *Connection) Subscribe(subject string, qgroup string, cb stan.MsgHandler, opts ...stan.SubscriptionOption) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sub := &subscription{
		subject: subject,
		group:   qgroup,
		cb:      cb,
		opts:    opts,
	}

	err = sub.subscribe(c)

	if err == nil {
		c.subs = append(c.subs, sub)
	}

	return
}

// Close closes the connection and forgets all subscriptions
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.subs = []*subscription{}

	return c.conn.Close()
}

// Publish publishes data to a specific subject
func (c *Connection) Publish(subject string, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.Publish(subject, data)
}

func (c *Connection) reconnect(ctx context.Context, reason error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	streamReconnectCtr.WithLabelValues(c.name, c.cfg.Name).Inc()

	c.log.Errorf("Reconnecting to NATS Stream after disconnection: %s", reason)

	c.connectSTAN(ctx)

	c.log.Infof("Resubscribing to %d subscriptions", len(c.subs))
	for _, sub := range c.subs {
		err := sub.subscribe(c)
		if err != nil {
			c.log.Errorf("Could not re-subscribe to %s: %s", sub.subject, err)
		}
	}
}

func (c *Connection) connectSTAN(ctx context.Context) {
	n := c.connectNATS(ctx)
	if n == nil {
		c.log.Errorf("%s NATS connection could not be established, cannot connect to the Stream", c.name)
		return
	}

	var err error
	try := 0

	for {
		try++

		reconf := func(_ stan.Conn, reason error) {
			errorCtr.WithLabelValues(c.name, c.cfg.Name).Inc()
			c.reconnect(ctx, reason)
		}

		c.conn, err = stan.Connect(c.id, c.name, stan.NatsConn(n), stan.SetConnectionLostHandler(reconf))
		if err != nil {
			c.log.Warnf("%s initial connection to the NATS Streaming broker cluster failed: %s", c.name, err)

			if ctx.Err() != nil {
				c.log.Errorf("%s initial connection canceled due to shut down", c.name)
				return
			}

			c.log.Infof("%s NATS Stream client failed connection attempt %d", c.name, try)

			if backoff.FiveSec.InterruptableSleep(ctx, try) != nil {
				return
			}

			continue
		}

		break
	}
}

func (c *Connection) connectNATS(ctx context.Context) (natsc *nats.Conn) {
	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.Name(c.name),
		nats.DisconnectHandler(c.disconCb),
		nats.ReconnectHandler(c.reconCb),
		nats.ClosedHandler(c.closedCb),
		nats.ErrorHandler(c.errorCb),
	}

	if c.tls {
		c.log.Debugf("Configuring TLS on NATS connection to %s", c.url)
		tlsc, err := c.cfg.SecurityProvider.TLSConfig()
		if err != nil {
			c.log.Errorf("Failed to configure TLS: %s", err)
			return nil
		}

		options = append(options, nats.Secure(tlsc))
	}

	var err error
	try := 0

	for {
		try++

		natsc, err = nats.Connect(c.url, options...)
		if err != nil {
			c.log.Warnf("%s initial connection to the NATS broker cluster (%s) failed: %s", c.name, c.url, err)

			if ctx.Err() != nil {
				c.log.Errorf("%s initial connection canceled due to shut down", c.name)
				return nil
			}

			s := backoff.FiveSec.Duration(try)
			c.log.Infof("%s NATS client sleeping %s after failed connection attempt %d", c.name, s, try)

			timer := time.NewTimer(s)

			select {
			case <-timer.C:
				continue
			case <-ctx.Done():
				c.log.Errorf("%s initial connection canceled due to shut down", c.name)
				return nil
			}
		}

		c.log.Infof("%s NATS client connected to %s", c.name, natsc.ConnectedUrl())

		break
	}

	return
}

func (c *Connection) disconCb(nc *nats.Conn) {
	err := nc.LastError()

	if err != nil {
		c.log.Warnf("%s NATS client connection got disconnected: %s", nc.Opts.Name, err)
	} else {
		c.log.Warnf("%s NATS client connection got disconnected", nc.Opts.Name)
	}
}

func (c *Connection) reconCb(nc *nats.Conn) {
	c.log.Warnf("%s NATS client reconnected after a previous disconnection, connected to %s", nc.Opts.Name, nc.ConnectedUrl())
	reconnectCtr.WithLabelValues(c.name, c.cfg.Name).Inc()
}

func (c *Connection) closedCb(nc *nats.Conn) {
	err := nc.LastError()

	if err != nil {
		c.log.Warnf("%s NATS client connection closed: %s", nc.Opts.Name, err)
	} else {
		c.log.Warnf("%s NATS client connection closed", nc.Opts.Name)
	}

	closedCtr.WithLabelValues(c.name, c.cfg.Name).Inc()
}

func (c *Connection) errorCb(nc *nats.Conn, sub *nats.Subscription, err error) {
	c.log.Errorf("%s NATS client on %s encountered an error: %s", nc.Opts.Name, nc.ConnectedUrl(), err)
	errorCtr.WithLabelValues(c.name, c.cfg.Name).Inc()
}
