package physicalmeasurement

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

func (c *PrometheusConfig) BuildFromPhysicalMeasurementMap(physicalMeasurementMap map[string][]string) {
	for deviceID, ips := range physicalMeasurementMap {
		cfg := ScrapeConfig{
			JobName: deviceID,
			StaticConfigs: []StaticConfig{
				{Targets: ips},
			},
			Params: map[string][]string{
				"module": {deviceID},
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
		c.ScrapeConfigs = append(c.ScrapeConfigs, cfg)
	}
}

func (c *PrometheusConfig) RemovePhysicalMeasurement(deviceId string, measurementIp string) {

}

func (c *PrometheusConfig) AddPhysicalMeasurement(deviceId string, measurementIp string) {

}
