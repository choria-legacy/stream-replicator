package ssl

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

type personality interface {
	SSLDir() (string, error)
	CAPath() (string, error)
	KeyPath() (string, error)
	CertPath() (string, error)
}

var cfg *config
var mu *sync.Mutex

func init() {
	mu = &sync.Mutex{}
}

func SSLContext() (*http.Transport, error) {
	tlsConfig, err := TLSConfig()
	if err != nil {
		return &http.Transport{}, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	return transport, nil
}

func TLSConfig() (tlsc *tls.Config, err error) {
	pub, _ := CertPath()
	pri, _ := KeyPath()
	ca, _ := CAPath()

	cert, err := tls.LoadX509KeyPair(pub, pri)
	if err != nil {
		return nil, fmt.Errorf("Could not load certificate %s and key %s: %s", pub, pri, err)
	}

	caCert, err := ioutil.ReadFile(ca)

	if err != nil {
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsc = &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	if !cfg.verify {
		tlsc.InsecureSkipVerify = true
	}

	tlsc.BuildNameToCertificate()

	return
}

func Check() (errors []string, ok bool) {
	if _, err := cfg.p.SSLDir(); err != nil {
		errors = append(errors, fmt.Sprintf("SSL Directory does not exist: %s", err))
		return errors, false
	}

	if c, err := cfg.p.CertPath(); err != nil {
		if _, err := os.Stat(c); err != nil {
			errors = append(errors, fmt.Sprintf("The Public Certificate %s does not exist", c))
		}

		if _, err = pemDecode(c); err != nil {
			errors = append(errors, fmt.Sprintf("The Public Certificate %s does not contain valid PEM data", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Could not determine Public Certificate path: %s", err))
	}

	if c, err := cfg.p.KeyPath(); err == nil {
		if _, err := os.Stat(c); err != nil {
			errors = append(errors, fmt.Sprintf("The Private Key %s does not exist", c))
		}

		if _, err = pemDecode(c); err != nil {
			errors = append(errors, fmt.Sprintf("The Private Key %s does not contain valid PEM data", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Could not determine Private Certificate path: %s", err))
	}

	if c, err := cfg.p.CAPath(); err == nil {
		if _, err := os.Stat(c); err != nil {
			errors = append(errors, fmt.Sprintf("The CA %s does not exist", c))
		}

		if _, err = pemDecode(c); err != nil {
			errors = append(errors, fmt.Sprintf("The CA %s does not contain valid PEM data", c))
		}
	} else {
		errors = append(errors, fmt.Sprintf("Could not determine CA path: %s", err.Error()))
	}

	if len(errors) == 0 {
		ok = true
	}

	return errors, ok
}

func SSLDir() (string, error) {
	return cfg.p.SSLDir()
}

func CAPath() (string, error) {
	return cfg.p.CAPath()
}

func KeyPath() (string, error) {
	return cfg.p.KeyPath()
}

func CertPath() (string, error) {
	return cfg.p.CertPath()
}

func CertificatePEM() (*pem.Block, error) {
	c, _ := CertPath()
	p, err := pemDecode(c)
	if err != nil {
		return nil, fmt.Errorf("Could not load Certificate data: %s", err)
	}

	return p, nil
}

func KeyPEM() (*pem.Block, error) {
	c, _ := KeyPath()
	p, err := pemDecode(c)
	if err != nil {
		return nil, fmt.Errorf("Could not load Key data: %s", err)
	}

	return p, nil
}

func CAPEM() (*pem.Block, error) {
	c, _ := CAPath()
	p, err := pemDecode(c)
	if err != nil {
		return nil, fmt.Errorf("Could not load CA data: %s", err)
	}

	return p, nil
}

func pemDecode(path string) (*pem.Block, error) {
	cdata, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("file not found %s: %s", path, err)
	}

	pb, _ := pem.Decode(cdata)
	if pb == nil {
		return pb, fmt.Errorf("failed to parse pem data in %s", path)
	}

	return pb, nil
}
