package physicalmeasurement

import (
	"os"

	"gopkg.in/yaml.v2"
)

type PrometheusConfig struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

type ScrapeConfig struct {
	JobName        string              `yaml:"job_name"`
	StaticConfigs  []StaticConfig      `yaml:"static_configs"`
	Params         map[string][]string `yaml:"params"`
	MetricsPath    string              `yaml:"metrics_path"`
	ScrapeInterval string              `yaml:"scrape_interval"`
	ScrapeTimeout  string              `yaml:"scrape_timeout"`
	RelabelConfigs []RelabelConfig     `yaml:"relabel_configs"`
}

type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

type RelabelConfig struct {
	SourceLabels []string `yaml:"source_labels,omitempty"`
	TargetLabel  string   `yaml:"target_label"`
	Replacement  string   `yaml:"replacement,omitempty"`
}

// Get rid of building -> do merging and create physical measurement map for single one.

func newPhysicalMeasurementConfig(device_id string, measurementIps []string) ScrapeConfig {
	cfg := ScrapeConfig{
		JobName: device_id,
		StaticConfigs: []StaticConfig{
			{Targets: measurementIps},
		},
		Params: map[string][]string{
			"module": {device_id},
			"auth":   {"public_v3"},
		},
		MetricsPath:    "/snmp",
		ScrapeInterval: "10s",
		ScrapeTimeout:  "5s",
		RelabelConfigs: []RelabelConfig{
			{SourceLabels: []string{"__address__"}, TargetLabel: "__param_target"},
			{SourceLabels: []string{"__param_target"}, TargetLabel: "instance"},
			{TargetLabel: "__addzress__", Replacement: "chantico-snmp:9116"},
		},
	}
	return cfg
}

func MergeWithPrometheusConfig(prometheus_yaml string, deviceId string, measurementIps []string) PrometheusConfig {
	cfg, _ := loadPrometheusConfig(prometheus_yaml)
	for _, scrape := range cfg.ScrapeConfigs {
		if scrape.JobName == deviceId {
			for _, target := range measurementIps {
				if !contains(scrape.StaticConfigs[0].Targets, target) {
					scrape.StaticConfigs[0].Targets = append(scrape.StaticConfigs[0].Targets, target)
				}
			}
			return *cfg
		}
	}
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, newPhysicalMeasurementConfig(deviceId, measurementIps))
	return *cfg
}

func RemoveFromPrometheusConfig(prometheus_yaml string, device_id string) PrometheusConfig {
	cfg, _ := loadPrometheusConfig(prometheus_yaml)
	newCfg := PrometheusConfig{}
	for _, scrape := range cfg.ScrapeConfigs {
		if scrape.JobName != device_id {
			newCfg.ScrapeConfigs = append(newCfg.ScrapeConfigs, scrape)
		}
	}
	return newCfg
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func loadPrometheusConfig(path string) (*PrometheusConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg PrometheusConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
