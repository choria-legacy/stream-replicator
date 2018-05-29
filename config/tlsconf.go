package config

import (
	"fmt"

	"github.com/choria-io/go-security"
	"github.com/choria-io/go-security/filesec"
	"github.com/choria-io/go-security/puppetsec"
	"github.com/sirupsen/logrus"
)

// TLSConf describes the TLS config for a NATS connection
type TLSConf struct {
	Identity string `json:"identity"`
	SSLDir   string `json:"ssl_dir"`
	Scheme   string `json:"scheme"`
	CA       string `json:"ca"`
	Cert     string `json:"cert"`
	Key      string `json:"key"`
}

// SecurityProvider creates a security provider for the given scheme
func (t *TLSConf) SecurityProvider() (security.Provider, error) {
	switch t.Scheme {
	case "puppet":
		return t.puppetSecurityProvider()
	case "file", "manual":
		return t.fileSecurityProvider()
	default:
		return nil, fmt.Errorf("unknown security scheme: %s", t.Scheme)
	}
}

func (t *TLSConf) puppetSecurityProvider() (security.Provider, error) {
	c := &puppetsec.Config{
		SSLDir:   t.SSLDir,
		Identity: t.Identity,
	}

	logger := logrus.New()

	return puppetsec.New(puppetsec.WithConfig(c), puppetsec.WithLog(logger.WithFields(logrus.Fields{"security": "puppet"})))
}

func (t *TLSConf) fileSecurityProvider() (security.Provider, error) {
	c := &filesec.Config{
		CA:          t.CA,
		Certificate: t.Cert,
		Key:         t.Key,
		Identity:    t.Identity,
	}

	logger := logrus.New()

	return filesec.New(filesec.WithConfig(c), filesec.WithLog(logger.WithFields(logrus.Fields{"security": "file"})))
}
