package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/choria-io/go-security/puppetsec"
	"github.com/choria-io/stream-replicator/config"
	"github.com/choria-io/stream-replicator/replicator"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	rep     = replicator.Copier{}
	cancel  func()
	ctx     context.Context
	version = "unknown"
	sha     = "unknown"

	cfile   string
	topic   string
	pidfile string

	enrollIdentity string
	enrollCA       string
	enrollDir      string

	reconn chan string
)

func Run() {
	app := kingpin.New("stream-replicator", "NATS Stream Topic Replicator")
	app.Author("R.I.Pienaar <rip@devco.net>")
	app.Version(version)

	replicate := app.Command("replicate", "Starts the Stream Replication process")
	replicate.Default()

	replicate.Flag("config", "Configuration file").StringVar(&cfile)
	replicate.Flag("topic", "Topic to replicate").Required().StringVar(&topic)
	replicate.Flag("pid", "Write running PID to a file").StringVar(&pidfile)

	enroll := app.Command("enroll", "Enrolls with a Puppet CA")
	enroll.Arg("identity", "Certificate Name to use when enrolling").StringVar(&enrollIdentity)
	enroll.Flag("ca", "Host and port for the Puppet CA in host:port format").Default("puppet:8140").StringVar(&enrollCA)
	enroll.Flag("dir", "Directory to write SSL configuration to").Required().StringVar(&enrollDir)

	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	reconn = make(chan string, 5)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	switch command {
	case "replicate":
		runReplicate()
	default:
		runEnroll()
	}
}

func runEnroll() {
	cfg := puppetsec.Config{
		Identity:   enrollIdentity,
		SSLDir:     enrollDir,
		DisableSRV: true,
	}

	re := regexp.MustCompile("^((([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])\\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\\-]*[A-Za-z0-9]))\\:(\\d+)$")

	if re.MatchString(enrollCA) {
		parts := strings.Split(enrollCA, ":")
		cfg.PuppetCAHost = parts[0]

		p, err := strconv.Atoi(parts[1])
		if err != nil {
			logrus.Fatalf("Could not enroll with the Puppet CA: %s", err)
		}

		cfg.PuppetCAPort = p
	}

	prov, err := puppetsec.New(puppetsec.WithConfig(&cfg), puppetsec.WithLog(logrus.WithField("provider", "puppet")))
	if err != nil {
		logrus.Fatalf("Could not enroll with the Puppet CA: %s", err)
	}

	wait, _ := time.ParseDuration("30m")

	err = prov.Enroll(ctx, wait, func(try int) { fmt.Printf("Attempting to download certificate for %s, try %d.\n", enrollIdentity, try) })
	if err != nil {
		logrus.Fatalf("Could not enroll with the Puppet CA: %s", err)
	}
}

func runReplicate() {
	done := make(chan int, 1)
	wg := &sync.WaitGroup{}

	err := config.Load(cfile)
	if err != nil {
		logrus.Fatalf("Could not parse configuration: %s", err)
		os.Exit(1)
	}

	configureLogging()

	topicconf, err := config.Topic(topic)
	if err != nil {
		logrus.Fatalf("Could not find a configuration for topic %s in the config file %s", topic, cfile)
		os.Exit(1)
	}

	go interruptHandler(wg)

	writePID(pidfile)

	logrus.Infof("Starting Choria Stream Replicator version %s for topic %s with configuration file %s", version, topic, cfile)

	startReplicator(ctx, wg, done, topicconf, topic)

	wg.Wait()
}

func writePID(pidfile string) {
	if pidfile == "" {
		return
	}

	err := ioutil.WriteFile(pidfile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	if err != nil {
		logrus.Fatalf("Could not write PID: %s", err)
		os.Exit(1)
	}
}

func interruptHandler(wg *sync.WaitGroup) {
	defer wg.Done()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case reason := <-reconn:
			logrus.Errorf("Restarting replicator after: %s", reason)

			// stops everything and sleep a bit to give state saves a bit of time etc
			cancel()
			time.Sleep(1 * time.Second)

			err := syscall.Exec(os.Args[0], os.Args, os.Environ())
			if err != nil {
				logrus.Errorf("Could not restart Stream Replicator: %s", err)
			}

		case sig := <-sigs:
			logrus.Infof("Shutting down on %s", sig)
			cancel()
			return

		case <-ctx.Done():
			return
		}
	}
}

func startReplicator(ctx context.Context, wg *sync.WaitGroup, done chan int, topic *config.TopicConf, topicname string) {
	err := rep.Setup(topicname, topic, reconn)
	if err != nil {
		logrus.Errorf("Could not configure Replicator: %s", err)
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
			logrus.Fatalf("Cannot open log file %s: %s", config.LogFile(), err)
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
