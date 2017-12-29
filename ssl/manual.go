package ssl

import (
	"path/filepath"
)

type manual struct {
	c *config
}

func (m *manual) SSLDir() (string, error) {
	return m.c.ssldir, nil
}

func (m *manual) CAPath() (string, error) {
	return filepath.Join(m.c.ssldir, m.c.ca), nil
}

func (m *manual) KeyPath() (string, error) {
	return filepath.Join(m.c.ssldir, m.c.key), nil
}

func (m *manual) CertPath() (string, error) {
	return filepath.Join(m.c.ssldir, m.c.cert), nil
}
