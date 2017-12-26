package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
)

type replications struct {
	Topics  map[string]TopicConf `json:"topics"`
	Debug   bool
	Verbose bool
	Logfile string `json:"logfile"`
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
		return fmt.Errorf("file %s could not be read: %s", file, err.Error())
	}

	j, err := yaml.YAMLToJSON(c)
	if err != nil {
		return fmt.Errorf("file %s could not be parsed: %s", file, err.Error())
	}

	err = json.Unmarshal(j, &config)
	if err != nil {
		return fmt.Errorf("Could not parse config file %s as YAML: %s", file, err.Error())
	}

	return nil
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
