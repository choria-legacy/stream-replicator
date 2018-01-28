package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/choria-io/stream-replicator/ssl"
	"github.com/ghodss/yaml"
)

type replications struct {
	Topics   map[string]TopicConf `json:"topics"`
	Debug    bool
	Verbose  bool
	Logfile  string   `json:"logfile"`
	TLS      *TLSConf `json:"tls"`
	StateDir string   `json:"state_dir"`
}

// TopicConf is the configuration for a specific topic
type TopicConf struct {
	Topic       string `json:"topic"`
	SourceURL   string `json:"source_url"`
	SourceID    string `json:"source_cluster_id"`
	TargetURL   string `json:"target_url"`
	TargetID    string `json:"target_cluster_id"`
	Workers     int
	Queued      bool
	QueueGroup  string `json:"queue_group"`
	Inspect     string
	MinAge      string `json:"age"`
	Name        string
	MonitorPort int `json:"monitor"`
	Advisory    *AdvisoryConf
}

// TLSConf describes the TLS config for a NATS connection
type TLSConf struct {
	SSLDir string `json:"ssl_dir"`
	Scheme string `json:"scheme"`
	CA     string `json:"ca"`
	Cert   string `json:"cert"`
	Key    string `json:"key"`
}

type AdvisoryConf struct {
	Target  string `json:"target"`
	Cluster string `json:"cluster" validate:"enum=source,target"`
	Age     string `json:"age"`
}

var config = replications{
	Topics: make(map[string]TopicConf),
}

// Load reads configuration from a YAML file
func Load(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("file %s not found", file)
	}

	c, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("config file could not be read: %s", err)
	}

	j, err := yaml.YAMLToJSON(c)
	if err != nil {
		return fmt.Errorf("file %s could not be parsed: %s", file, err)
	}

	err = json.Unmarshal(j, &config)
	if err != nil {
		return fmt.Errorf("Could not parse config file %s as YAML: %s", file, err)
	}

	if config.TLS != nil {
		ssl.Configure(config.TLS.Scheme, config.TLS.Options()...)
	}

	return nil
}

// StateDirectory is where a cache of seen data will be saved when configured
func StateDirectory() string {
	return config.StateDir
}

// TLS determines if TLS is configured
func TLS() bool {
	return config.TLS != nil
}

// Debug enables debug logging
func Debug() bool {
	return config.Debug
}

// Verbose enables verbose logging
func Verbose() bool {
	return config.Verbose
}

// LogFile is the file to log to, STDOUT when empty
func LogFile() string {
	return config.Logfile
}

// Topic is the configuration for a specific topic from the file
func Topic(name string) (TopicConf, error) {
	t, ok := config.Topics[name]
	if !ok {
		return TopicConf{}, fmt.Errorf("Unknown topic configuration: %s", name)
	}

	return t, nil
}

// Options return options that configure the ssl personalities
func (t *TLSConf) Options() (opts []ssl.Option) {
	if t.SSLDir != "" {
		opts = append(opts, ssl.Directory(t.SSLDir))
	}

	if t.CA != "" {
		opts = append(opts, ssl.CA(t.CA))
	}

	if t.Cert != "" {
		opts = append(opts, ssl.Cert(t.Cert))
	}

	if t.Key != "" {
		opts = append(opts, ssl.Key(t.Key))
	}

	return opts
}
