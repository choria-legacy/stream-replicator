package test

import (
	gnatsd "github.com/nats-io/gnatsd/server"
	stan "github.com/nats-io/nats-streaming-server/server"
)

func RunNatsServer(host string, port int) *gnatsd.Server {
	if host == "" {
		host = "localhost"
	}

	opts := &gnatsd.Options{
		Host: host,
		Port: port,
		// NoLog:          true,
		NoSigs:         true,
		MaxControlLine: 256,
		Debug:          true,
		LogFile:        "/tmp/log",
	}

	s := gnatsd.New(opts)
	s.ConfigureLogger()
	go s.Start()

	return s
}

func RunLeftServer(url string) *stan.StanServer {
	sopts := stan.GetDefaultOptions()
	sopts.ID = "left"
	sopts.MaxChannels = 10
	sopts.NATSServerURL = url

	s, err := stan.RunServerWithOpts(sopts, nil)
	if err != nil {
		panic(err)
	}

	return s
}

func RunRightServer(url string) *stan.StanServer {
	sopts := stan.GetDefaultOptions()
	sopts.ID = "right"
	sopts.MaxChannels = 10
	sopts.NATSServerURL = url

	s, err := stan.RunServerWithOpts(sopts, nil)
	if err != nil {
		panic(err)
	}

	return s
}
