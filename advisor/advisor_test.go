package advisor

import (
	"context"
	"os"
	"sync"
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
	RunSpecs(t, "Advisor")
}

var _ = Describe("Advisor", func() {
	var (
		ctx      context.Context
		wg       *sync.WaitGroup
		log      *logrus.Entry
		goodconf *config.TopicConf
	)

	BeforeEach(func() {
		ctx = context.Background()
		wg = &sync.WaitGroup{}
		logrus.SetOutput(os.Stdout)
		log = logrus.WithField("test", true)
		logrus.SetLevel(logrus.FatalLevel)
		reset()

		goodconf = &config.TopicConf{
			SourceID:  "left",
			SourceURL: "nats://localhost:34222",
			TargetID:  "right",
			TargetURL: "nats://localhost:44222",
			Name:      "testing",
			Inspect:   "sender",
			MinAge:    "2h",
			Advisory: &config.AdvisoryConf{
				Age:     "15m",
				Cluster: "source",
				Target:  "test.target",
			},
		}
	})

	var _ = Describe("connect", func() {
		BeforeEach(func() {
			err := Configure(false, goodconf)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should connect to the source server when configured", func() {
			ns := conntest.RunNatsServer("localhost", 34222)
			defer ns.Shutdown()

			if !ns.ReadyForConnections(10 * time.Second) {
				panic("NATS server did not become ready after 10 seconds")
			}

			left := conntest.RunLeftServer("nats://localhost:34222")
			defer left.Shutdown()

			conf.Advisory.Cluster = "source"
			connect(ctx)

			Expect(conn.NatsConn().ConnectedUrl()).To(Equal("nats://localhost:34222"))
		})

		It("Should connect to the target server when configured", func() {
			ns := conntest.RunNatsServer("localhost", 44222)
			defer ns.Shutdown()

			if !ns.ReadyForConnections(10 * time.Second) {
				panic("NATS server did not become ready after 10 seconds")
			}

			right := conntest.RunRightServer("nats://localhost:44222")
			defer right.Shutdown()

			conf.Advisory.Cluster = "target"
			connect(ctx)

			Expect(conn.NatsConn().ConnectedUrl()).To(Equal("nats://localhost:44222"))
		})
	})

	var _ = Describe("Configure", func() {
		It("Should be a noop when no advisory is configured", func() {
			Configure(false, &config.TopicConf{})
			Expect(configured).To(BeFalse())
		})

		It("Should handle invalid times", func() {
			c := &config.TopicConf{
				Advisory: &config.AdvisoryConf{
					Age: "x",
				},
			}

			err := Configure(true, c)
			Expect(err).To(MatchError("age cannot be parsed as a duration: time: invalid duration x"))
			Expect(configured).To(BeFalse())
		})

		It("Should mark it as configured on success", func() {
			err := Configure(true, goodconf)
			Expect(err).ToNot(HaveOccurred())
			Expect(configured).To(BeTrue())
			Expect(name).To(Equal("testing_advisor"))
		})
	})

	var _ = Describe("Record", func() {
		It("Should noop when not configured", func() {
			Expect(seen).To(BeEmpty())
			Record("test")
			Expect(seen).To(BeEmpty())
		})

		It("Should record the sender", func() {
			err := Configure(true, goodconf)
			Expect(err).ToNot(HaveOccurred())
			Expect(seen).To(BeEmpty())
			Record("test")
			Expect(seen).To(HaveLen(1))
		})

		It("Should send advisories if this is a previously advised about sender", func() {
			err := Configure(true, goodconf)
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(HaveLen(0))

			Record("test")
			Expect(seen).To(HaveLen(1))
			advised["test"] = time.Now().UTC()

			Record("test")

			Expect(out).To(HaveLen(1))
			Expect(advised).To(HaveLen(0))

			msg := <-out
			Expect(msg.Value).To(Equal("test"))
			Expect(msg.Event).To(Equal(Recovery))
		})
	})

	var _ = Describe("advise", func() {
		It("Should advise all senders not seen in the configured time", func() {
			err := Configure(true, goodconf)
			Expect(err).ToNot(HaveOccurred())

			Expect(seen).To(BeEmpty())

			seen["old"] = time.Now().Add(-1 * time.Hour)
			seen["expired"] = time.Now().Add(-3 * time.Hour)
			seen["new"] = time.Now()

			Expect(out).To(HaveLen(0))

			advise()

			Expect(out).To(HaveLen(2))

			msg := <-out
			Expect(msg.Value).To(Equal("old"))
			Expect(msg.Event).To(Equal(Timeout))

			msg = <-out
			Expect(msg.Value).To(Equal("expired"))
			Expect(msg.Event).To(Equal(Expired))

			_, found := advised["old"]
			Expect(found).To(BeTrue())

			_, found = advised["expired"]
			Expect(found).To(BeFalse())
		})
	})
})
