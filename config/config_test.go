package config

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFederation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config")
}

var _ = Describe("Config", func() {
	var _ = Describe("Load", func() {
		It("Should fail for nonexisting files", func() {
			err := Load("nonexisting")
			Expect(err).To(MatchError("file nonexisting not found"))
		})

		It("Should fail for unreadable files", func() {
			s, _ := os.Stat("testdata/unreadable.yaml")
			m := s.Mode()

			os.Chmod("testdata/unreadable.yaml", os.FileMode(0))
			defer os.Chmod("testdata/unreadable.yaml", m)

			err := Load("testdata/unreadable.yaml")
			Expect(err).To(MatchError("config file could not be read: open testdata/unreadable.yaml: permission denied"))

		})

		It("Should parse good files", func() {
			err := Load("testdata/good.yaml")
			Expect(err).ToNot(HaveOccurred())

			_, ok := config.Topics["dc1_cmdb"]
			Expect(ok).To(BeTrue())

			_, ok = config.Topics["dc3_cmdb"]
			Expect(ok).To(BeTrue())

			Expect(config.Topics["dc1_cmdb"].Advisory.Target).To(Equal("sr.advisories.cmdb"))
			Expect(config.Debug).To(BeTrue())
			Expect(config.Verbose).To(BeFalse())

			Expect(config.TLS).ToNot(BeNil())
			Expect(config.SecurityProvider).ToNot(BeNil())

			Expect(TLS()).To(BeTrue())
			Expect(Debug()).To(BeTrue())
			Expect(Verbose()).To(BeFalse())
			Expect(LogFile()).To(Equal("/tmp/log"))
			Expect(Topic("dc1_cmdb")).To(Equal(config.Topics["dc1_cmdb"]))
		})
	})
})
