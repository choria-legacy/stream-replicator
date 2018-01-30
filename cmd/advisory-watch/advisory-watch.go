package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/choria-io/stream-replicator/advisor"
	"github.com/fatih/color"

	"github.com/nats-io/go-nats-streaming"
	"github.com/sirupsen/logrus"

	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/connector"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	cfile  string
	topic  string
	name   string
	conf   config.TopicConf
	err    error
	conn   stan.Conn
	log    *logrus.Entry
	ctx    context.Context
	cancel func()
	debug  bool

	all    bool
	filter string
	since  time.Duration
)

func connect() {
	var c *connector.Connection

	if conf.Advisory.Cluster == "source" {
		c = connector.New(name, false, connector.Source, conf, log)
	} else {
		c = connector.New(name, false, connector.Target, conf, log)
	}

	conn = c.Connect(ctx)
}

func parseCLI() {
	kingpin.Flag("config", "Configuration file").ExistingFileVar(&cfile)
	kingpin.Flag("topic", "Topic to observe advisories for").Required().StringVar(&topic)
	kingpin.Flag("name", "Client name for the connection").StringVar(&name)
	kingpin.Flag("debug", "Enables debug logging").BoolVar(&debug)
	kingpin.Flag("all", "Retrieve all advsirories").BoolVar(&all)
	kingpin.Flag("since", "Retrieve advisories since a certain duration").DurationVar(&since)
	kingpin.Flag("filter", "Filters the advisories to display using a regular expression").StringVar(&filter)

	kingpin.Parse()

	if cfile == "" {
		cfile = "/etc/stream-replicator/sr.yaml"
	}

	config.Load(cfile)

	logrus.SetLevel(logrus.WarnLevel)

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	log = logrus.WithFields(logrus.Fields{"topic": topic})

	conf, err = config.Topic(topic)
	if err != nil {
		kingpin.Fatalf("%s", err)
	}

	if conf.Advisory == nil {
		kingpin.Fatalf("Advisories are not configured for topic %s", topic)
	}

	if name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			kingpin.Fatalf("Cannot determine hostname, specify --name manually: %s", err)
		}

		name = fmt.Sprintf("sr_advisory_watcher_%d_%s", os.Geteuid(), strings.Replace(hostname, ".", "_", -1))
	}
}

func interruptHandler() {
	ctx, cancel = context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	waiter := func() {
		for {
			select {
			case sig := <-sigs:
				log.Infof("Shutting down on %s", sig)
				cancel()
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}

	go waiter()
}

func viewf(msg *stan.Msg) {
	advisory := advisor.AgeAdvisoryV1{}
	json.Unmarshal(msg.Data, &advisory)

	if filter != "" {
		show, err := regexp.MatchString(filter, advisory.Value)
		if err != nil {
			kingpin.Fatalf("Cannot apply filter %s: %s", filter, err)
		}

		if !show {
			return
		}
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	switch advisory.Event {
	case advisor.Recovery:
		fmt.Printf("%s %s: %s: recovered\n", green("✓"), time.Unix(advisory.Timestamp, 0), advisory.Value)
	case advisor.Timeout:
		fmt.Printf("%s %s: %s: timeout after not being seen for %ds\n", yellow("↓"), time.Unix(advisory.Timestamp, 0), advisory.Value, advisory.Age)
	case advisor.Expired:
		fmt.Printf("%s %s: %s: expired after not being seen for %ds\n", red("✗"), time.Unix(advisory.Timestamp, 0), advisory.Value, advisory.Age)
	}
}

func main() {
	parseCLI()

	interruptHandler()

	connect()

	opts := []stan.SubscriptionOption{}

	if all && int64(since) > 0 {
		kingpin.Fatalf("Cannot honor both all and since")
	}

	if all {
		opts = append(opts, stan.StartAtSequence(0))
	}

	if int64(since) > 0 {
		opts = append(opts, stan.StartAtTimeDelta(since))
	}

	log.Infof("Subscribing to %s", conf.Advisory.Target)

	conn.Subscribe(conf.Advisory.Target, viewf, opts...)

	select {
	case <-ctx.Done():
		return
	}
}
