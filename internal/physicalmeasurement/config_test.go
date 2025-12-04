package physicalmeasurement

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestMakeScrapeConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{
		"foo": {"10.0.0.1", "10.0.0.2"},
	}
	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)

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
`)

	checkEquality(t, cfg, expected)
}

func TestAddToScrapeConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{}

	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)
	cfg.AddPhysicalMeasurement("foo", "10.0.0.1")
	cfg.AddPhysicalMeasurement("foo", "10.0.0.2")

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
`)

	checkEquality(t, cfg, expected)
}

func TestDuplicateAddToScrapeConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{}

	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)
	cfg.AddPhysicalMeasurement("foo", "10.0.0.1")
	cfg.AddPhysicalMeasurement("foo", "10.0.0.1")
	cfg.AddPhysicalMeasurement("foo", "10.0.0.2")

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
`)

	checkEquality(t, cfg, expected)
}

func TestMultipleDevicesAddToScrapeConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{}

	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)
	cfg.AddPhysicalMeasurement("foo", "10.0.0.1")
	cfg.AddPhysicalMeasurement("bar", "10.0.0.1")

	expected := []byte(`
scrape_configs:
  - job_name: foo
    static_configs:
      - targets:
        - 10.0.0.1
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
        - 10.0.0.1
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

func TestRemoveEmptyConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{}

	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)
	cfg.RemovePhysicalMeasurement("foo", "10.0.0.1")
	expected := []byte(`
scrape_configs: []
`)
	checkEquality(t, cfg, expected)
}

func TestRemoveOneEntryConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{}

	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)
	cfg.AddPhysicalMeasurement("foo", "10.0.0.1")
	cfg.RemovePhysicalMeasurement("foo", "10.0.0.1")
	expected := []byte(`
scrape_configs: []
`)
	checkEquality(t, cfg, expected)
}

func TestRemoveConfig(t *testing.T) {
	physicalMeasurementMap := map[string][]string{}

	cfg := PrometheusConfig{}
	cfg.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)
	cfg.AddPhysicalMeasurement("foo", "10.0.0.1")
	cfg.AddPhysicalMeasurement("bar", "10.0.0.2")
	cfg.RemovePhysicalMeasurement("foo", "10.0.0.1")
	expected := []byte(`
scrape_configs:
  - job_name: bar
    static_configs:
      - targets:
        - 10.0.0.2
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
