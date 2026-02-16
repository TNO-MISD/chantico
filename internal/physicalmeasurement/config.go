package physicalmeasurement

import (
	"os"

	"gopkg.in/yaml.v2"
)

type GlobalConfig struct {
	ScrapeInterval     string `yaml:"scrape_interval,omitempty"`
	EvaluationInterval string `yaml:"evaluation_interval,omitempty"`
}

type PrometheusConfig struct {
	Global        *GlobalConfig  `yaml:"global,omitempty"`
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

func CreatePrometheusConfig(device_id string, measurementIps []string) PrometheusConfig {
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
	prometheusCfg := PrometheusConfig{
		ScrapeConfigs: []ScrapeConfig{cfg},
	}
	return prometheusCfg
}

func MergeWithPrometheusConfig(configs []PrometheusConfig) PrometheusConfig {
	// Map to deduplicate jobs by name
	jobMap := make(map[string]*ScrapeConfig)

	for _, config := range configs {
		for _, scrape := range config.ScrapeConfigs {
			if existing, ok := jobMap[scrape.JobName]; ok {
				// Job exists - merge targets
				for _, staticConfig := range scrape.StaticConfigs {
					for _, newTarget := range staticConfig.Targets {
						if !contains(existing.StaticConfigs[0].Targets, newTarget) {
							existing.StaticConfigs[0].Targets = append(
								existing.StaticConfigs[0].Targets,
								newTarget,
							)
						}
					}
				}
			} else {
				newScrape := scrape
				jobMap[scrape.JobName] = &newScrape
			}
		}
	}

	var allScrapeConfigs []ScrapeConfig
	for _, scrape := range jobMap {
		allScrapeConfigs = append(allScrapeConfigs, *scrape)
	}

	return PrometheusConfig{
		ScrapeConfigs: allScrapeConfigs,
	}
}

// Helper function to check if a slice contains a string
func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func LoadPrometheusConfig(path string) (*PrometheusConfig, error) {
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
