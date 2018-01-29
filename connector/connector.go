package connector

import (
	"context"
	"time"

	"github.com/choria-io/stream-replicator/backoff"
	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/ssl"
	nats "github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"
)

// Connection holds a connection to NATS Streaming
type Connection struct {
	url  string
	log  *logrus.Entry
	conn stan.Conn
	name string
	cfg  config.TopicConf
	id   string
	tls  bool
}

// Direction indicates which of the connectors to connect to
type Direction uint8

// Source indicates a connection to the source should be made
const Source = Direction(0)

// Target indicates a connection to the target should be made
const Target = Direction(1)

// New creates a new connector
func New(name string, tls bool, dir Direction, cfg config.TopicConf, logger *logrus.Entry) *Connection {
	c := Connection{
		url:  cfg.TargetURL,
		log:  logger,
		name: name,
		id:   cfg.TargetID,
		tls:  tls,
		cfg:  cfg,
	}

	if dir == Source {
		c.url = cfg.SourceURL
		c.id = cfg.SourceID
	}

	return &c
}

// Connect connects to the configured stream
func (c *Connection) Connect(ctx context.Context) stan.Conn {
	c.conn = c.connectSTAN(ctx, c.id, c.name, c.url)

	return c.conn
}

func (c *Connection) connectSTAN(ctx context.Context, cid string, name string, urls string) stan.Conn {
	n := c.connectNATS(ctx, name, urls)
	if n == nil {
		c.log.Errorf("%s NATS connection could not be established, cannot connect to the Stream", name)
		return nil
	}

	var err error
	var conn stan.Conn
	try := 0

	for {
		try++

		conn, err = stan.Connect(cid, name, stan.NatsConn(n))
		if err != nil {
			c.log.Warnf("%s initial connection to the NATS Streaming broker cluster failed: %s", name, err)

			if ctx.Err() != nil {
				c.log.Errorf("%s initial connection cancelled due to shut down", name)
				return nil
			}

			c.log.Infof("%s NATS Stream client failed connection attempt %d", name, try)

			if backoff.FiveSec.InterruptableSleep(ctx, try) != nil {
				return nil
			}

			continue
		}

		break
	}

	return conn
}

func (c *Connection) connectNATS(ctx context.Context, name string, urls string) (natsc *nats.Conn) {
	options := []nats.Option{
		nats.MaxReconnects(-1),
		nats.Name(name),
		nats.DisconnectHandler(c.disconCb),
		nats.ReconnectHandler(c.reconCb),
		nats.ClosedHandler(c.closedCb),
		nats.ErrorHandler(c.errorCb),
	}

	if c.tls {
		tlsc, err := ssl.TLSConfig()
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

		natsc, err = nats.Connect(urls, options...)
		if err != nil {
			c.log.Warnf("%s initial connection to the NATS broker cluster failed: %s", name, err)

			if ctx.Err() != nil {
				c.log.Errorf("%s initial connection cancelled due to shut down", name)
				return nil
			}

			s := backoff.FiveSec.Duration(try)
			c.log.Infof("%s NATS client sleeping %s after failed connection attempt %d", name, s, try)

			timer := time.NewTimer(s)

			select {
			case <-timer.C:
				continue
			case <-ctx.Done():
				c.log.Errorf("%s initial connection cancelled due to shut down", name)
				return nil
			}
		}

		c.log.Infof("%s NATS client connected to %s", name, natsc.ConnectedUrl())

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
