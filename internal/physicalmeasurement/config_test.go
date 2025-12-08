package physicalmeasurement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

const (
	exampleYaml = `
scrape_configs:
  - job_name: foo
    static_configs:
      - targets:
        - 10.0.0.1
        - 10.0.0.2
    params:
      module:
        - foo
      auth:
        - public_v3
    metrics_path: /snmp
    scrape_interval: 10s
    scrape_timeout: 5s
    relabel_configs:
      - source_labels: ["__address__"]
        target_label: "__param_target"
      - source_labels: ["__param_target"]
        target_label: "instance"
      - target_label: "__addzress__"
        replacement: chantico-snmp:9116
`
)

func writeConfigToFile(t *testing.T, config []byte, filename string) (*string, error) {
	filePath := filepath.Join(t.TempDir(), filename)
	err := os.WriteFile(filePath, config, 0644)

	if err != nil {
		return nil, err
	}
	return &filePath, nil
}

func TestMakeScrapeConfig(t *testing.T) {
	device_id := "foo"
	measurement_ips := []string{"10.0.0.1", "10.0.0.2"}
	cfg := PrometheusConfig{}
	scrape := CreatePhysicalMeasurementConfig(device_id, measurement_ips)
	cfg.ScrapeConfigs = append(cfg.ScrapeConfigs, scrape)

	checkEquality(t, cfg, []byte(exampleYaml))
}

func TestMergeToConfigExistingJob(t *testing.T) {
	yml, _ := writeConfigToFile(t, []byte(exampleYaml), "test_cfg.yaml")
	scrape_cfg := CreatePhysicalMeasurementConfig("foo", []string{"10.0.0.3", "10.0.0.4"})
	cfg := MergeWithPrometheusConfig(*yml, scrape_cfg)

	expected := []byte(`
scrape_configs:
  - job_name: foo
    static_configs:
      - targets:
        - 10.0.0.1
        - 10.0.0.2
        - 10.0.0.3
        - 10.0.0.4
    params:
      module:
        - foo
      auth:
        - public_v3
    metrics_path: /snmp
    scrape_interval: 10s
    scrape_timeout: 5s
    relabel_configs:
      - source_labels: ["__address__"]
        target_label: "__param_target"
      - source_labels: ["__param_target"]
        target_label: "instance"
      - target_label: "__addzress__"
        replacement: chantico-snmp:9116
`)

	checkEquality(t, cfg, expected)
}

func TestMergeToConfigNewJob(t *testing.T) {
	yml, _ := writeConfigToFile(t, []byte(exampleYaml), "test_cfg.yaml")
	scrape_cfg := CreatePhysicalMeasurementConfig("bar", []string{"10.0.0.3", "10.0.0.4"})
	cfg := MergeWithPrometheusConfig(*yml, scrape_cfg)

	expected := []byte(`
scrape_configs:
  - job_name: foo
    static_configs:
      - targets:
        - 10.0.0.1
        - 10.0.0.2
    params:
      module:
        - foo
      auth:
        - public_v3
    metrics_path: /snmp
    scrape_interval: 10s
    scrape_timeout: 5s
    relabel_configs:
      - source_labels: ["__address__"]
        target_label: "__param_target"
      - source_labels: ["__param_target"]
        target_label: "instance"
      - target_label: "__addzress__"
        replacement: chantico-snmp:9116
  - job_name: bar
    static_configs:
      - targets:
        - 10.0.0.3
        - 10.0.0.4
    params:
      module:
        - bar
      auth:
        - public_v3
    metrics_path: /snmp
    scrape_interval: 10s
    scrape_timeout: 5s
    relabel_configs:
      - source_labels: ["__address__"]
        target_label: "__param_target"
      - source_labels: ["__param_target"]
        target_label: "instance"
      - target_label: "__addzress__"
        replacement: chantico-snmp:9116
`)

	checkEquality(t, cfg, expected)
}

func checkEquality(t *testing.T, actual PrometheusConfig, expected []byte) {
	yamlBytes, err := yaml.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal yaml: %v", err)
	}

	var expectedObj map[string]interface{}
	err = yaml.Unmarshal(expected, &expectedObj)
	if err != nil {
		t.Fatalf("failed to unmarshal expected yaml: %v", err)
	}

	var actualObj map[string]interface{}
	err = yaml.Unmarshal(yamlBytes, &actualObj)
	if err != nil {
		t.Fatalf("failed to unmarshal actual yaml: %v", err)
	}

	if diff := deepEqualDiff(expectedObj, actualObj); diff != "" {
		t.Errorf("YAML mismatch:\n%s", diff)
	}
}

func deepEqualDiff(a, b interface{}) string {
	if cmp.Equal(a, b) {
		return ""
	}
	return cmp.Diff(a, b)
}
