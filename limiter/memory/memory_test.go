package memory

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
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
		wg     *sync.WaitGroup
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		wg = &sync.WaitGroup{}

		config.Load("testdata/empty.yaml")

		logrus.SetOutput(os.Stdout)
		logrus.SetLevel(logrus.FatalLevel)

		m = Limiter{
			log: logrus.WithFields(logrus.Fields{}),
		}
	})

	AfterEach(func() {
		cancel()
		wg.Wait()
		os.Remove("testdata/test.json")
	})

	var _ = Describe("Configure", func() {
		It("Should configure the key and age", func() {
			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")

			Expect(m.key).To(Equal("k"))
			Expect(m.age).To(Equal(time.Duration(1 * time.Minute)))
			Expect(m.processed).To(BeEmpty())
		})
	})

	var _ = Describe("shouldProcess", func() {
		BeforeEach(func() {
			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")
		})

		It("Should be true for empty values", func() {
			Expect(m.shouldProcess("")).To(BeTrue())
		})

		It("Should be true the first time its seen", func() {
			Expect(m.shouldProcess("test")).To(BeTrue())
		})

		It("Should be false when recently been seen", func() {
			m.processed["test"] = time.Now()
			Expect(m.shouldProcess("test")).To(BeFalse())
		})

		It("Should correctly detect when a update is needed based on age", func() {
			m.processed["test"] = time.Now().Add(-59 * time.Second)
			Expect(m.shouldProcess("test")).To(BeFalse())

			m.processed["test"] = time.Now().Add(-121 * time.Second)
			Expect(m.shouldProcess("test")).To(BeTrue())
		})
	})

	var _ = Describe("scrub", func() {
		It("Should delete only old entries", func() {
			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")

			m.processed["new"] = time.Now()
			m.processed["old"] = time.Now().Add(-3 * time.Minute)

			m.scrub()
			Expect(m.processed).ToNot(HaveKey("old"))
			Expect(m.processed).To(HaveKey("new"))
		})
	})

	var _ = Describe("Configure", func() {
		It("Should set the statefile to empty by default", func() {
			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")

			Expect(m.statefile).To(BeEmpty())
		})

		It("Should set the statefile if configured", func() {
			config.Load("testdata/stateconfig.yaml")

			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")

			Expect(m.statefile).To(Equal("testdata/test.json"))
		})
	})

	var _ = Describe("writeCache", func() {
		BeforeEach(func() {
			config.Load("testdata/stateconfig.yaml")
			os.Remove("testdata/test.json")

			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")
		})

		It("Should not write when unconfigured", func() {
			err := m.writeCache()
			Expect(err).ToNot(HaveOccurred())
			Expect(m.statefile).ToNot(BeAnExistingFile())
		})

		It("Should write the cache", func() {
			m.processed["test"] = time.Now()
			err := m.writeCache()
			Expect(err).ToNot(HaveOccurred())
			Expect(m.statefile).To(BeAnExistingFile())

			s := make(map[string]time.Time)
			d, _ := ioutil.ReadFile(m.statefile)
			err = json.Unmarshal(d, &s)

			Expect(err).ToNot(HaveOccurred())
			Expect(s["test"].Unix()).To(Equal(m.processed["test"].Unix()))
		})
	})

	var _ = Describe("readCache", func() {
		It("Should attempt to read the cache when configured", func() {
			config.Load("testdata/stateconfig.yaml")
			os.Remove("testdata/test.json")

			m.Configure(ctx, wg, "k", time.Duration(1*time.Minute), "test")

			m.processed["test"] = time.Now()
			m.writeCache()
			Expect(m.statefile).To(BeAnExistingFile())

			u := m.processed["test"].Unix()
			m.processed = make(map[string]time.Time)
			m.readCache()
			Expect(m.processed).ToNot(BeEmpty())
			Expect(u).To(Equal(m.processed["test"].Unix()))
		})
	})
})
