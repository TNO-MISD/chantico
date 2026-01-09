package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
)

const (
	StateInit      = "init"
	StateRunning   = "Running"
	StateDelete    = "Delete"
	StateCompleted = "Completed"
	StateFailed    = "Failed"
)

func UpdateState(
	physicalMeasurement *chantico.PhysicalMeasurement,
) {
	if physicalMeasurement == nil {
		return
	}
	if physicalMeasurement.Status.UpdateGeneration == 0 {
		physicalMeasurement.Status.UpdateGeneration = 1
	}

	// Covers lifecycle related changes
	switch {
	case physicalMeasurement.Status.UpdateGeneration < physicalMeasurement.ObjectMeta.Generation:
		physicalMeasurement.Status.State = StateInit
		break
	case physicalMeasurement.ObjectMeta.GetDeletionTimestamp() != nil:
		physicalMeasurement.Status.State = StateDelete
		break
	}

	switch physicalMeasurement.Status.State {
	case "", StateInit:
		physicalMeasurement.Status.State = StateInit
		physicalMeasurement.Status.UpdateGeneration = physicalMeasurement.ObjectMeta.Generation
		return
	case StateRunning, StateDelete, StateFailed:
		return
	default:
		physicalMeasurement.Status.State = StateFailed
		return
	}
}
