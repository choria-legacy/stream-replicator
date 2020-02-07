package config

import (
	security "github.com/choria-io/go-choria/providers/security"
)

// TopicConf is the configuration for a specific topic
type TopicConf struct {
	Topic            string        `json:"topic"`
	SourceURL        string        `json:"source_url"`
	SourceID         string        `json:"source_cluster_id"`
	TargetURL        string        `json:"target_url"`
	TargetID         string        `json:"target_cluster_id"`
	Workers          int           `json:"workers"`
	Queued           bool          `json:"queued"`
	QueueGroup       string        `json:"queue_group"`
	Inspect          string        `json:"inspect"`
	UpdateFlag       string        `json:"update_flag"`
	MinAge           string        `json:"age"`
	Name             string        `json:"name"`
	MonitorPort      int           `json:"monitor"`
	Advisory         *AdvisoryConf `json:"advisory"`
	TLSc             *TLSConf      `json:"tls"`
	DisableTargetTLS bool          `json:"disable_target_tls"`
	DisableSourceTLS bool          `json:"disable_source_tls"`

	SecurityProvider security.Provider `json:"-"`
}

// TLS determines if the topic has a TLS configuration set
func (t *TopicConf) TLS() bool {
	return t.TLSc != nil
}
