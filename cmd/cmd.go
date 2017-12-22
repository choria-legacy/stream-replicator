package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/choria-io/stream-replicator/replicator"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	debug   bool
	verbose bool
	rep     = replicator.Copier{}
	cancel  func()
	ctx     context.Context
	mport   int
	// seq     int
	// all     bool
	// last    bool
	// startd  time.Duration
)

func Run() {
	app := kingpin.New("natsreplicate", "NATS Stream Topic Replicator")
	app.Author("R.I.Pienaar <rip@devco.net>")
	app.UsageTemplate(kingpin.SeparateOptionalFlagsUsageTemplate)

	app.Flag("name", "Replicator name").StringVar(&rep.Name)
	app.Flag("topic", "Topic to replicate").Required().StringVar(&rep.Topic)
	app.Flag("from", "NATS Stream cluster host URL(s) to replicate from").Required().StringVar(&rep.From)
	app.Flag("from-id", "Cluster ID for the source cluster").Required().StringVar(&rep.FromID)
	app.Flag("to", "NATS Stream cluster host URL(s) to replicate to").Required().StringVar(&rep.To)
	app.Flag("to-id", "Cluster ID for the target cluster").Required().StringVar(&rep.ToID)
	app.Flag("workers", "Number of workers to start").Default("1").IntVar(&rep.Workers)
	app.Flag("queued", "Subscribe to a queue group, true when workers > 0").BoolVar(&rep.Queued)
	app.Flag("verbose", "Set verbose logging").Default("false").BoolVar(&verbose)
	app.Flag("debug", "Set debug logging").Default("false").BoolVar(&debug)
	app.Flag("monitor", "Port to listen for Prometheus requests on /metrics").IntVar(&mport)
	// app.Flag("seq", "Start replicating from a specific sequence").IntVar(&seq)
	// app.Flag("all", "Replicate all available messages").BoolVar(&all)
	// app.Flag("last", "Start replicating from the last message").BoolVar(&last)
	// app.Flag("since", "Start delivering messages from a specific time").DurationVar(&startd)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	done := make(chan int, 1)
	wg := &sync.WaitGroup{}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	configureLogging()

	go interruptHandler()

	if mport > 0 {
		go setupPrometheus()
	}

	startReplicator(ctx, wg, done)

	wg.Wait()
}

func setupPrometheus() {
	http.Handle("/metrics", promhttp.Handler())
	logrus.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", mport), nil))
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

func startReplicator(ctx context.Context, wg *sync.WaitGroup, done chan int) {
	err := rep.Setup()
	if err != nil {
		logrus.Errorf("Could not configure Replicator: %s", err.Error())
		return
	}

	wg.Add(1)
	go rep.Run(ctx, wg)
}

func configureLogging() {
	logrus.SetLevel(logrus.InfoLevel)

	if verbose {
		logrus.SetLevel(logrus.InfoLevel)
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
}
