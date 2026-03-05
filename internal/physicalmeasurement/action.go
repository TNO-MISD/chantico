package physicalmeasurement

import (
	"bytes"
	chantico "chantico/api/v1alpha1"
	ph "chantico/internal/patch"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"

	vol "chantico/internal/volumes"

	"go.yaml.in/yaml/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PrometheusTarget struct {
	Labels    map[string]string `json:"labels"`
	Health    string            `json:"health"`
	LastError string            `json:"lastError"`
}

type ActionFunctionType int

const (
	ActionFunctionIO = iota
	ActionFunctionPure
)

type ActionResult struct {
	*ctrl.Result
	ph.PatchType
}

type ActionFunction struct {
	Type ActionFunctionType
	Pure func(
		*chantico.PhysicalMeasurement,
	) *ActionResult
	IO func(
		context.Context,
		client.Client,
		*chantico.PhysicalMeasurement,
	) *ActionResult
}

var ActionMap = map[State][]ActionFunction{
	StateInit: {
		ActionFunction{Type: ActionFunctionPure, Pure: InitializeFinalizer},
		ActionFunction{Type: ActionFunctionPure, Pure: WriteConfigFile},
		ActionFunction{Type: ActionFunctionPure, Pure: CombineConfigFiles},
		ActionFunction{Type: ActionFunctionPure, Pure: ReloadPrometheus},
	},
	StateRunning: {
		ActionFunction{Type: ActionFunctionPure, Pure: CheckEndpointHealth},
	},
	StateRunningWithWarning: {
		ActionFunction{Type: ActionFunctionPure, Pure: CheckEndpointHealth},
	},
	StateDelete: {
		ActionFunction{Type: ActionFunctionPure, Pure: DeleteConfigFile},
		ActionFunction{Type: ActionFunctionPure, Pure: CombineConfigFiles},
		ActionFunction{Type: ActionFunctionPure, Pure: ReloadPrometheus},
		ActionFunction{Type: ActionFunctionPure, Pure: UpdateFinalizer},
	},
	StateFailed: {},
}

func InitializeFinalizer(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	if slices.Contains(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer) {
		return nil
	}
	physicalMeasurement.ObjectMeta.Finalizers = append(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer)
	log.Printf("ADDED FINALIZER: %#v\n", physicalMeasurement.ObjectMeta.Finalizers)
	return &ActionResult{PatchType: ph.PatchResource}
}

func UpdateFinalizer(
	physicalMeasurement *chantico.PhysicalMeasurement,
) *ActionResult {
	if physicalMeasurement.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}
	accumulator := []string{}
	for _, f := range physicalMeasurement.ObjectMeta.Finalizers {
		if f != chantico.PhysicalMeasurementFinalizer {
			accumulator = append(accumulator, f)
		}
	}
	physicalMeasurement.ObjectMeta.Finalizers = accumulator
	return &ActionResult{PatchType: ph.PatchResource}
}

func ExecuteActions(
	ctx context.Context,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,
	patch *ph.PatchHelper,
) *ActionResult {
	var result *ActionResult = nil
	actionFunctions := ActionMap[State(physicalMeasurement.Status.State)]
	for i, actionFunction := range actionFunctions {
		log.Printf("Start step %d, status: %s\n", i, physicalMeasurement.Status.State)
		switch actionFunction.Type {
		case ActionFunctionPure:
			result = actionFunction.Pure(physicalMeasurement)
		case ActionFunctionIO:
			result = actionFunction.IO(ctx, c, physicalMeasurement)
		}

		if result != nil {
			patch.Patch(result.PatchType)
			if result.Result != nil || physicalMeasurement.Status.State == StateFailed {
				break
			}
		}
	}
	return result
}

func CheckEndpointHealth(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	url := fmt.Sprintf("http://%s:%s/api/v1/targets", os.Getenv("CHANTICO_PROMETHEUS_SERVICE_HOST"), os.Getenv("CHANTICO_PROMETHEUS_SERVICE_PORT"))

	state, errMsg := queryTargetHealth(url, physicalMeasurement.Spec.Ip, physicalMeasurement.Spec.MeasurementDevice)
	physicalMeasurement.Status.State = state
	physicalMeasurement.Status.ErrorMessage = errMsg
	return &ActionResult{
		Result:    &ctrl.Result{RequeueAfter: chantico.EndpointRequeueDelay},
		PatchType: ph.PatchResourceStatus,
	}
}

func queryTargetHealth(url, targetIP string, auth string) (string, string) {
	resp, err := http.Get(url)
	if err != nil {
		return StateRunningWithWarning, fmt.Sprintf("Cannot reach Prometheus: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			ActiveTargets []PrometheusTarget `json:"activeTargets"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return StateRunningWithWarning, fmt.Sprintf("Failed to parse targets: %v", err)
	}

	return matchTargetHealth(result.Data.ActiveTargets, targetIP, auth)
}

func matchTargetHealth(targets []PrometheusTarget, targetIP, auth string) (string, string) {
	for _, t := range targets {
		if t.Labels["instance"] == targetIP && t.Labels["job"] == auth {
			if t.Health == "up" {
				return StateRunning, ""
			}
			return StateRunningWithWarning, fmt.Sprintf("Cannot reach target: %s", t.LastError)
		}
	}
	return StateRunningWithWarning, "Target not yet registered in Prometheus"
}

func WriteConfigFile(
	physicalMeasurement *chantico.PhysicalMeasurement,
) *ActionResult {
	cfg := CreatePrometheusConfig(physicalMeasurement.Spec.MeasurementDevice, []string{physicalMeasurement.Spec.Ip})

	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	configPath := volumePath + "/prometheus/yml/" + physicalMeasurement.Name + ".yml"
	yamlBytes, _ := yaml.Marshal(cfg)
	err := os.WriteFile(configPath, yamlBytes, 0644)
	if err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to write Prometheus config file: %v", err)
		return &ActionResult{PatchType: ph.PatchResourceStatus}
	}
	physicalMeasurement.Status.State = StateRunning
	return &ActionResult{PatchType: ph.PatchResourceStatus}
}

func CombineConfigFiles(
	_ *chantico.PhysicalMeasurement,
) *ActionResult {
	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	configDir := volumePath + "/prometheus/yml"
	prometheusYmlPath := configDir + "/prometheus.yml"

	existingConfig, _ := LoadPrometheusConfig(prometheusYmlPath)

	entries, err := os.ReadDir(configDir)
	if err != nil {
		log.Printf("Failed to read config directory: %v", err)
	}

	var configs []PrometheusConfig

	for _, entry := range entries {
		// Skip directories and prometheus.yml itself
		if entry.IsDir() || entry.Name() == "prometheus.yml" {
			continue
		}

		filePath := configDir + "/" + entry.Name()
		config, err := LoadPrometheusConfig(filePath)
		if err != nil {
			log.Printf("Failed to load config file %s: %v", entry.Name(), err)
			continue
		}

		configs = append(configs, *config)
	}

	combinedConfig := MergeWithPrometheusConfig(configs)

	// Preserve global config from existing prometheus.yml
	if existingConfig != nil && existingConfig.Global != nil {
		combinedConfig.Global = existingConfig.Global
	}

	combinedYaml, err := yaml.Marshal(combinedConfig)
	if err != nil {
		log.Printf("Failed to marshal combined config: %v", err)
	}

	if err := os.WriteFile(prometheusYmlPath, combinedYaml, 0644); err != nil {
		log.Printf("Failed to write prometheus.yml: %v", err)
	}

	log.Printf("Combined scrape configs into %s", prometheusYmlPath)
	return &ActionResult{}
}

func DeleteConfigFile(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	configPath := volumePath + "/prometheus/yml/" + physicalMeasurement.Name + ".yml"

	log.Printf("Deleting Prometheus config for %s\n", physicalMeasurement.Name)

	err := os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to delete config file: %v", err)
		return &ActionResult{PatchType: ph.PatchResourceStatus}
	}

	return &ActionResult{PatchType: ph.PatchResourceStatus}
}

func ReloadPrometheus(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	address := fmt.Sprintf("http://%s:%s/-/reload", os.Getenv("CHANTICO_PROMETHEUS_SERVICE_HOST"), os.Getenv("CHANTICO_PROMETHEUS_SERVICE_PORT"))

	resp, err := http.Post(address, "application/json", bytes.NewBuffer([]byte{}))
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		log.Printf("Failed to reload Prometheus: %v", err)

		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = fmt.Sprintf("Prometheus reload failed with status: %v", err)
		return &ActionResult{PatchType: ph.PatchResourceStatus}
	}
	defer resp.Body.Close()
	log.Println("Prometheus reloaded successfully.")
	return &ActionResult{PatchType: ph.PatchResourceStatus}
}
