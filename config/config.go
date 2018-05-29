package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/choria-io/go-security"
	"github.com/ghodss/yaml"
)

type replications struct {
	Topics   map[string]*TopicConf `json:"topics"`
	Debug    bool
	Verbose  bool
	Logfile  string   `json:"logfile"`
	TLS      *TLSConf `json:"tls"`
	StateDir string   `json:"state_dir"`

	SecurityProvider security.Provider
}

// TopicConf is the configuration for a specific topic
type TopicConf struct {
	Topic       string        `json:"topic"`
	SourceURL   string        `json:"source_url"`
	SourceID    string        `json:"source_cluster_id"`
	TargetURL   string        `json:"target_url"`
	TargetID    string        `json:"target_cluster_id"`
	Workers     int           `json:"workers"`
	Queued      bool          `json:"queued"`
	QueueGroup  string        `json:"queue_group"`
	Inspect     string        `json:"inspect"`
	UpdateFlag  string        `json:"update_flag"`
	MinAge      string        `json:"age"`
	Name        string        `json:"name"`
	MonitorPort int           `json:"monitor"`
	Advisory    *AdvisoryConf `json:"advisory"`
	TLS         *TLSConf      `json:"tls"`

	SecurityProvider security.Provider `json:"-"`
}

// AdvisoryConf configures an advisory target
type AdvisoryConf struct {
	Target  string `json:"target"`
	Cluster string `json:"cluster" validate:"enum=source,target"`
	Age     string `json:"age"`
}

var config = replications{
	Topics: make(map[string]*TopicConf),
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
		return fmt.Errorf("could not parse config file %s as YAML: %s", file, err)
	}

	if config.TLS != nil {
		config.SecurityProvider, err = config.TLS.SecurityProvider()
		if err != nil {
			return fmt.Errorf("could not configure system SSL: %s", err)
		}

	}

	for _, t := range config.Topics {
		t.SecurityProvider = config.SecurityProvider

		if t.TLS == nil {
			t.TLS = config.TLS
		}

		if t.TLS != nil {
			t.SecurityProvider, err = t.TLS.SecurityProvider()
			if err != nil {
				return fmt.Errorf("could not configure topic %s SSL: %s", t.Name, err)
			}
		}
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
func Topic(name string) (*TopicConf, error) {
	t, ok := config.Topics[name]
	if !ok {
		return nil, fmt.Errorf("unknown topic configuration: %s", name)
	}

	return t, nil
}
