package limiter

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFederation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Memory")
}

var _ = Describe("Memeory", func() {
	var m Memory

	BeforeEach(func() {
		m = Memory{}
	})

	var _ = Describe("Configure", func() {
		It("Should configure the key and age", func() {
			m.Configure("k", time.Duration(1*time.Minute))
			Expect(m.key).To(Equal("k"))
			Expect(m.age).To(Equal(time.Duration(1 * time.Minute)))
			Expect(m.seen).To(BeEmpty())
		})
	})

	var _ = Describe("shouldProcess", func() {
		BeforeEach(func() {
			m.Configure("k", time.Duration(1*time.Minute))
		})

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

	var _ = Describe("scrubber", func() {
		It("Should delete only old entries", func() {
			m.Configure("k", time.Duration(1*time.Minute))

			m.seen["new"] = time.Now()
			m.seen["old"] = time.Now().Add(-61 * time.Second)
			m.scrubber()
			Expect(m.seen).ToNot(HaveKey("old"))
			Expect(m.seen).To(HaveKey("new"))
		})
	})
})
