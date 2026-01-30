package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
	ph "chantico/internal/patch"
	"context"
	"log"
	"os"
	"slices"

	vol "chantico/internal/volumes"

	"go.yaml.in/yaml/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ActionFunctionType int

const (
	ActionFunctionIO = iota
	ActionFunctionPure
)

type ActionResult struct {
	ctrl.Result
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

var ActionMap = map[string][]ActionFunction{
	StateInit: {
		ActionFunction{Type: ActionFunctionPure, Pure: InitializeFinalizer},
		ActionFunction{Type: ActionFunctionPure, Pure: WritePrometheusConfig},
		ActionFunction{Type: ActionFunctionPure, Pure: ReloadPrometheus},
	},
	StateRunning: {},
	StateDelete: {
		ActionFunction{Type: ActionFunctionPure, Pure: DeletePhysicalMeasurementConfig},
		ActionFunction{Type: ActionFunctionPure, Pure: ReloadPrometheus},
	},
	StateCompleted: {},
	StateFailed:    {},
}

func InitializeFinalizer(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	if slices.Contains(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer) {
		return nil
	}
	physicalMeasurement.ObjectMeta.Finalizers = append(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer)
	log.Printf("Added finalizer: %#v\n", physicalMeasurement.ObjectMeta.Finalizers)
	return nil
}

// RemoveFinalizer

func ExecuteActions(
	ctx context.Context,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,

) *ActionResult {
	var result *ActionResult = nil
	actionFunctions := ActionMap[string(physicalMeasurement.Status.State)]
	for i, actionFunction := range actionFunctions {
		log.Printf("Start step %d, status: %s\n", i, physicalMeasurement.Status.State)
		switch actionFunction.Type {
		case ActionFunctionPure:
			result = actionFunction.Pure(physicalMeasurement)
		case ActionFunctionIO:
			result = actionFunction.IO(ctx, c, physicalMeasurement)
		}
		if result != nil {
		}
		if physicalMeasurement.Status.State == StateFailed {
			break
		}
	}
	return result
}

func WritePrometheusConfig(
	physicalMeasurement *chantico.PhysicalMeasurement,
) *ActionResult {
	// physicalMeasurement.Status.UpdateGeneration = physicalMeasurement.ObjectMeta.Generation
	cfg := CreatePrometheusConfig(physicalMeasurement.Spec.MeasurementDevice, physicalMeasurement.Spec.ResourceIds)

	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	configPath := volumePath + "/prometheus/yml/" + physicalMeasurement.Name + ".yml"

	yamlBytes, _ := yaml.Marshal(cfg)
	err := os.WriteFile(configPath, yamlBytes, 0644)
	if err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("%v", err)
		return &ActionResult{}
	}

	return &ActionResult{}
}

func DeletePhysicalMeasurementConfig(physicalMeasurement *chantico.PhysicalMeasurement) *ActionResult {
	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	configPath := volumePath + "/prometheus/yml/" + physicalMeasurement.Name + ".yml"

	log.Printf("Deleting Prometheus config for PhysicalMeasurement %s\n", physicalMeasurement.Name)

	err := os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to delete config file: %v", err)
		return &ActionResult{}
	}

	return nil
}

func ReloadPrometheus(_ *chantico.PhysicalMeasurement) *ActionResult {

	return nil
}
