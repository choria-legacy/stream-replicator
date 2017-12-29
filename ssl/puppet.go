package ssl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type puppet struct {
	c *config
}

func (p *puppet) SSLDir() (string, error) {
	if p.c.ssldir != "" {
		return p.c.ssldir, nil
	}

	if os.Getuid() == 0 {
		path, err := p.puppetSetting("ssldir")
		if err != nil {
			return "", err
		}

		// store it so future calls to this wil not call out to Puppet again
		p.c.ssldir = path

		return path, nil
	}

	if os.Getenv("HOME") == "" {
		return "", fmt.Errorf("cannot determine home dir, set HOME environment or configure ssldir")
	}

	return filepath.Join(os.Getenv("HOME"), ".puppetlabs", "etc", "puppet", "ssl"), nil
}

func (p *puppet) CAPath() (string, error) {
	ssl, err := p.SSLDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "certs", "ca.pem"), nil
}

func (p *puppet) KeyPath() (string, error) {
	ssl, err := p.SSLDir()
	if err != nil {
		return "", err
	}

	cn, err := p.certname()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "private_keys", fmt.Sprintf("%s.pem", cn)), nil
}

func (p *puppet) CertPath() (string, error) {
	ssl, err := p.SSLDir()
	if err != nil {
		return "", err
	}

	cn, err := p.certname()
	if err != nil {
		return "", err
	}

	return filepath.Join(ssl, "certs", fmt.Sprintf("%s.pem", cn)), nil
}

func (p *puppet) puppetSetting(setting string) (string, error) {
	args := []string{"apply", "--configprint", setting}

	out, err := exec.Command("puppet", args...).Output()
	if err != nil {
		return "", err
	}

	return strings.Replace(string(out), "\n", "", -1), nil
}

func (p *puppet) certname() (string, error) {
	certname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	if os.Getuid() != 0 {
		if u, ok := os.LookupEnv("USER"); ok {
			certname = fmt.Sprintf("%s.mcollective", u)
		}
	}

	return certname, nil
}
