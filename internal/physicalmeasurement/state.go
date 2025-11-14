package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
)

const (
	StateRunning   = "Running"
	StateCompleted = "Completed"
	StateFailed    = "Failed"
	StateReloaded  = "Reloaded"
)

func GetState(
	physicalMeasurement *chantico.PhysicalMeasurement,
	job *batchv1.Job,
	deployment *appsv1.Deployment,
) string {
	if physicalMeasurement == nil {
		return StateFailed
	}

	switch physicalMeasurement.Status.State {
	case "":
		return StateRunning
	default:
		panic("Not implemented yet")
	}
}
