package connector

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/choria-io/stream-replicator/config"
	conntest "github.com/choria-io/stream-replicator/connector/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFederation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connector")
}

var _ = Describe("Connector", func() {
	var log *logrus.Entry
	var conf *config.TopicConf

	BeforeSuite(func() {
		logrus.SetOutput(os.Stdout)
		log = logrus.WithField("test", true)
		logrus.SetLevel(logrus.FatalLevel)

		conf = &config.TopicConf{
			SourceID:  "left",
			SourceURL: "nats://localhost:34222",
			TargetID:  "right",
			TargetURL: "nats://localhost:44222",
			Name:      "testing",
			Inspect:   "sender",
			Advisory: &config.AdvisoryConf{
				Age:     "15m",
				Cluster: "source",
				Target:  "test.target",
			},
		}
	})

	var _ = Describe("New", func() {
		It("Should configure the correct direction", func() {
			c := New("testcon", true, Source, conf, log)
			Expect(c.cfg).To(Equal(conf))
			Expect(c.id).To(Equal("left"))
			Expect(c.url).To(Equal("nats://localhost:34222"))
			Expect(c.name).To(Equal("testcon"))
			Expect(c.tls).To(BeTrue())
		})
	})

	var _ = Describe("Connect", func() {
		It("Should connect to the stream", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ns := conntest.RunNatsServer("localhost", 34222)
			defer ns.Shutdown()

			if !ns.ReadyForConnections(10 * time.Second) {
				panic("NATS server did not become ready after 10 seconds")
			}

			left := conntest.RunLeftServer("nats://localhost:34222")
			defer left.Shutdown()

			c := New("testcon", false, Source, conf, log)
			con := c.Connect(ctx)
			defer con.Close()
		})
	})
})
