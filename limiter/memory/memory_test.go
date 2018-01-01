package memory

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/choria-io/stream-replicator/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestFederation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Limiter/Memory")
}

var _ = Describe("Limiter/Memory", func() {
	var (
		m      Limiter
		ctx    context.Context
		cancel func()
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		config.Load("testdata/empty.yaml")

		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.FatalLevel)
		m = Limiter{
			log: logrus.WithFields(logrus.Fields{}),
		}
		m.Configure(ctx, "k", time.Duration(1*time.Minute), "test")
	})

	AfterEach(func() {
		cancel()
		os.Remove("testdata/test.json")
	})

	var _ = Describe("Configure", func() {
		It("Should configure the key and age", func() {
			Expect(m.key).To(Equal("k"))
			Expect(m.age).To(Equal(time.Duration(1 * time.Minute)))
			Expect(m.seen).To(BeEmpty())
		})
	})

	var _ = Describe("shouldProcess", func() {
		It("Should be true for empty values", func() {
			Expect(m.shouldProcess("")).To(BeTrue())
		})

		It("Should be true the first time its seen", func() {
			Expect(m.shouldProcess("test")).To(BeTrue())
		})

		It("Should be false when recently been seen", func() {
			m.seen["test"] = time.Now()
			Expect(m.shouldProcess("test")).To(BeFalse())
		})

		It("Should correctly detect when a update is needed based on age", func() {
			m.seen["test"] = time.Now().Add(-59 * time.Second)
			Expect(m.shouldProcess("test")).To(BeFalse())

			m.seen["test"] = time.Now().Add(-61 * time.Second)
			Expect(m.shouldProcess("test")).To(BeTrue())
		})
	})

	var _ = Describe("scrub", func() {
		It("Should delete only old entries", func() {
			m.seen["new"] = time.Now()
			m.seen["old"] = time.Now().Add((-61 * time.Second) - (10 * time.Minute))
			m.scrub()
			Expect(m.seen).ToNot(HaveKey("old"))
			Expect(m.seen).To(HaveKey("new"))
		})
	})

	var _ = Describe("Configure", func() {
		It("Should set the statefile to empty by default", func() {
			Expect(m.statefile).To(BeEmpty())
		})

		It("Should set the statefile if configured", func() {
			config.Load("testdata/stateconfig.yaml")

			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
			m.Configure(ctx, "k", time.Duration(1*time.Minute), "test")

			Expect(m.statefile).To(Equal("testdata/test.json"))
		})
	})

	var _ = Describe("writeCache", func() {
		BeforeEach(func() {
			config.Load("testdata/stateconfig.yaml")
			os.Remove("testdata/test.json")

			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()

			m.Configure(ctx, "k", time.Duration(1*time.Minute), "test")
		})

		It("Should not write when unconfigured", func() {
			m.writeCache()
			Expect(m.statefile).ToNot(BeAnExistingFile())
		})

		It("Should write the cache", func() {
			m.seen["test"] = time.Now()
			m.writeCache()
			Expect(m.statefile).To(BeAnExistingFile())

			s := make(map[string]time.Time)
			d, _ := ioutil.ReadFile(m.statefile)
			err := json.Unmarshal(d, &s)

			Expect(err).ToNot(HaveOccurred())
			Expect(s["test"].Unix()).To(Equal(m.seen["test"].Unix()))
		})
	})

	var _ = Describe("readCache", func() {
		It("Should attempt to read the cache when configured", func() {
			config.Load("testdata/stateconfig.yaml")
			os.Remove("testdata/test.json")

			cancel()
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()

			m.Configure(ctx, "k", time.Duration(1*time.Minute), "test")

			m.seen["test"] = time.Now()
			m.writeCache()
			Expect(m.statefile).To(BeAnExistingFile())

			u := m.seen["test"].Unix()
			m.seen = make(map[string]time.Time)
			m.readCache()
			Expect(m.seen).ToNot(BeEmpty())
			Expect(u).To(Equal(m.seen["test"].Unix()))
		})
	})
})
