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

type StateActions struct {
	ActionFunctions []ActionFunction
	PatchType       ph.PatchType
}

type ActionFunction struct {
	Type ActionFunctionType
	Pure func(
		*chantico.PhysicalMeasurement,
	) *ctrl.Result
	IO func(
		context.Context,
		client.Client,
		*chantico.PhysicalMeasurement,
	) *ctrl.Result
}

var ActionMap = map[string]StateActions{
	StateInit: {
		ActionFunctions: []ActionFunction{
			{Type: ActionFunctionPure, Pure: InitializeFinalizer},
			{Type: ActionFunctionPure, Pure: WritePrometheusConfig},
			{Type: ActionFunctionPure, Pure: ReloadPrometheus},
		},
		PatchType: ph.PatchObject,
	},
	StateRunning: {
		ActionFunctions: []ActionFunction{},
		PatchType:       ph.PatchObjectStatus,
	},
	StateDelete: {
		ActionFunctions: []ActionFunction{
			{Type: ActionFunctionPure, Pure: DeletePhysicalMeasurementConfig},
			{Type: ActionFunctionPure, Pure: ReloadPrometheus},
		},
		PatchType: ph.PatchObject,
	},
	StateCompleted: {},
	StateFailed:    {},
}

func InitializeFinalizer(physicalMeasurement *chantico.PhysicalMeasurement) *ctrl.Result {
	if slices.Contains(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer) {
		return nil
	}
	log.Printf("Adding finalizer to PhysicalMeasurement %s\n", physicalMeasurement.Name)
	physicalMeasurement.ObjectMeta.Finalizers = append(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer)
	return nil
}

// RemoveFinalizer

func ExecuteActions(
	ctx context.Context,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,

) ph.ResultToPatch {
	var patchResult ph.ResultToPatch
	stateActions := ActionMap[string(physicalMeasurement.Status.State)]
	patchResult.PatchType = stateActions.PatchType
	for _, actionFunction := range stateActions.ActionFunctions {
		switch actionFunction.Type {
		case ActionFunctionPure:
			patchResult.Result = actionFunction.Pure(physicalMeasurement)
		case ActionFunctionIO:
			patchResult.Result = actionFunction.IO(ctx, c, physicalMeasurement)
		}
		if patchResult.Result != nil || physicalMeasurement.Status.State == StateFailed {
			break
		}
	}
	return patchResult
}

func WritePrometheusConfig(
	physicalMeasurement *chantico.PhysicalMeasurement,
) *ctrl.Result {
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
		return &ctrl.Result{}
	}

	return &ctrl.Result{}
}

func DeletePhysicalMeasurementConfig(physicalMeasurement *chantico.PhysicalMeasurement) *ctrl.Result {
	volumePath := os.Getenv(vol.ChanticoVolumeLocationEnv)
	configPath := volumePath + "/prometheus/yml/" + physicalMeasurement.Name + ".yml"

	log.Printf("Deleting Prometheus config for PhysicalMeasurement %s\n", physicalMeasurement.Name)

	err := os.Remove(configPath)
	if err != nil && !os.IsNotExist(err) {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		log.Printf("Failed to delete config file: %v", err)
		return &ctrl.Result{}
	}

	return nil
}

func ReloadPrometheus(_ *chantico.PhysicalMeasurement) *ctrl.Result {

	return nil
}
