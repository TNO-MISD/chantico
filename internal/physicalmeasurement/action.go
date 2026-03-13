package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
	ph "chantico/internal/patch"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"

	vol "chantico/internal/volumes"

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

const prometheusTargetsDir = "prometheus/targets"

// ActionMap defines the actions to execute for each state.
// With file_sd_configs, Prometheus automatically watches the target files
// for changes — no explicit reload or config merging is needed.
var ActionMap = map[State][]ActionFunction{
	StateInit: {
		ActionFunction{Type: ActionFunctionPure, Pure: InitializeFinalizer},
		ActionFunction{Type: ActionFunctionPure, Pure: WriteTargetFile},
	},
	StateRunning: {
		ActionFunction{Type: ActionFunctionPure, Pure: CheckEndpointHealth},
	},
	StateRunningWithWarning: {
		ActionFunction{Type: ActionFunctionPure, Pure: CheckEndpointHealth},
	},
	StateDelete: {
		ActionFunction{Type: ActionFunctionPure, Pure: DeleteTargetFile},
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

// WriteTargetFile writes a file_sd_configs JSON target file for this PhysicalMeasurement.
// The file is written to prometheus/targets/<name>.json.
// Prometheus automatically detects changes to these files and updates its scrape targets.
func WriteTargetFile(
	physicalMeasurement *chantico.PhysicalMeasurement,
) *ActionResult {
	target := CreateFileSDTarget(physicalMeasurement.Spec.MeasurementDevice, physicalMeasurement.Spec.Ip)

	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	targetsDir := filepath.Join(volumePath, prometheusTargetsDir)
	if err := os.MkdirAll(targetsDir, 0777); err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to create targets directory: %v", err)
		return &ActionResult{PatchType: ph.PatchResourceStatus}
	}

	targetPath := filepath.Join(targetsDir, physicalMeasurement.Name+".json")
	if err := WriteFileSDTargets(targetPath, []FileSDTarget{target}); err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to write target file: %v", err)
		return &ActionResult{PatchType: ph.PatchResourceStatus}
	}

	log.Printf("Wrote file_sd target file %s for device %s\n", targetPath, physicalMeasurement.Spec.MeasurementDevice)
	physicalMeasurement.Status.State = StateRunning
	return &ActionResult{PatchType: ph.PatchResourceStatus}
}

// DeleteTargetFile removes the file_sd_configs target file for this PhysicalMeasurement.
// Prometheus will automatically stop scraping the removed targets.
func DeleteTargetFile(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	targetPath := filepath.Join(volumePath, prometheusTargetsDir, physicalMeasurement.Name+".json")

	log.Printf("Deleting target file for %s\n", physicalMeasurement.Name)

	err := os.Remove(targetPath)
	if err != nil && !os.IsNotExist(err) {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to delete target file: %v", err)
		return &ActionResult{PatchType: ph.PatchResourceStatus}
	}

	return &ActionResult{PatchType: ph.PatchResourceStatus}
}
