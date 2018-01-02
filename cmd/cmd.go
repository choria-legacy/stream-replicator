package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/replicator"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	rep     = replicator.Copier{}
	cancel  func()
	ctx     context.Context
	version string
	sha     string
)

func Run() {
	app := kingpin.New("stream-replicator", "NATS Stream Topic Replicator")
	app.Author("R.I.Pienaar <rip@devco.net>")
	app.Version(version)

	cfile := ""
	topic := ""
	pidfile := ""

	app.Flag("config", "Configuration file").StringVar(&cfile)
	app.Flag("topic", "Topic to replicate").Required().StringVar(&topic)
	app.Flag("pid", "Write running PID to a file").StringVar(&pidfile)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	done := make(chan int, 1)
	wg := &sync.WaitGroup{}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	err := config.Load(cfile)
	if err != nil {
		logrus.Fatalf("Could not parse configuration: %s", err.Error())
		os.Exit(1)
	}

	configureLogging()

	topicconf, err := config.Topic(topic)
	if err != nil {
		logrus.Fatalf("Could not find a configuration for topic %s in the config file %s", topic, cfile)
		os.Exit(1)
	}

	go interruptHandler()

	writePID(pidfile)
	startReplicator(ctx, wg, done, topicconf)

	wg.Wait()
}

func writePID(pidfile string) {
	if pidfile == "" {
		return
	}

	err := ioutil.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	if err != nil {
		logrus.Fatalf("Could not write PID: %s", err.Error())
		os.Exit(1)
	}
}

func interruptHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigs:
			logrus.Infof("Shutting down on %s", sig)
			cancel()
		case <-ctx.Done():
			return
		}
	}
}

func startReplicator(ctx context.Context, wg *sync.WaitGroup, done chan int, topic config.TopicConf) {
	err := rep.Setup(topic)
	if err != nil {
		logrus.Errorf("Could not configure Replicator: %s", err.Error())
		return
	}

	if topic.MonitorPort > 0 {
		go rep.SetupPrometheus(topic.MonitorPort)
	}

	wg.Add(1)
	go rep.Run(ctx, wg)
}

func configureLogging() {
	if config.LogFile() != "" {
		logrus.SetFormatter(&logrus.JSONFormatter{})

		file, err := os.OpenFile(config.LogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			logrus.Fatalf("Cannot open log file %s: %s", config.LogFile(), err.Error())
			os.Exit(1)
		}

		logrus.SetOutput(file)
	}

	logrus.SetLevel(logrus.InfoLevel)

	if config.Verbose() {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if config.Debug() {
		logrus.SetLevel(logrus.DebugLevel)
	}
}
