package connector

import (
	"github.com/nats-io/go-nats-streaming"
)

type subscription struct {
	subject string
	group   string
	cb      stan.MsgHandler
	opts    []stan.SubscriptionOption
	sub     stan.Subscription
}

func (s *subscription) subscribe(c *Connection) (err error) {
	if s.group == "" {
		c.log.Infof("Subscribing to subject %s", s.subject)
		s.sub, err = c.conn.Subscribe(c.cfg.Topic, s.cb, s.opts...)
	} else {
		c.log.Infof("Subscribing to subject %s in group %s", s.subject, s.group)
		s.sub, err = c.conn.QueueSubscribe(c.cfg.Topic, s.group, s.cb, s.opts...)
	}

	return
}
