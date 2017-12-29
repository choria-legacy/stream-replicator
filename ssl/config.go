package ssl

type config struct {
	ca     string
	key    string
	cert   string
	ssldir string
	verify bool
	p      personality
}

type Option func(*config) error

func Configure(personality string, opts ...Option) error {
	mu.Lock()
	defer mu.Unlock()

	cfg := &config{}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return err
		}
	}

	switch personality {
	case "choria":
		cfg.p = &puppet{cfg}
	default:
		cfg.p = &manual{cfg}
	}

	return nil
}

func CA(path string) Option {
	return func(c *config) error {
		c.ca = path
		return nil
	}
}

func Key(path string) Option {
	return func(c *config) error {
		c.key = path
		return nil
	}
}

func Cert(path string) Option {
	return func(c *config) error {
		c.cert = path
		return nil
	}
}

func Directory(path string) Option {
	return func(c *config) error {
		c.ssldir = path
		return nil
	}
}

func Verify() Option {
	return func(c *config) error {
		c.verify = true
		return nil
	}
}

func UnVerified() Option {
	return func(c *config) error {
		c.verify = false
		return nil
	}
}
