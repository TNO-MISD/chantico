package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
	"slices"

	batchv1 "k8s.io/api/batch/v1"
)

type State string

const (
	StateInit      = "init"
	StateRunning   = "Running"
	StateDelete    = "Delete"
	StateCompleted = "Completed"
	StateFailed    = "Failed"
)

// TODO delete reference to job since all actions are not interacting with the cluster.
func UpdateState(
	physicalMeasurement *chantico.PhysicalMeasurement, job *batchv1.Job,
) {
	if physicalMeasurement == nil {
		return
	}
	if physicalMeasurement.Status.UpdateGeneration == 0 {
		physicalMeasurement.Status.UpdateGeneration = 1
	}

	if !slices.Contains(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer) {
		physicalMeasurement.Status.State = StateInit
		return
	}

	// Covers lifecycle related changes
	isDeleted := physicalMeasurement.ObjectMeta.GetDeletionTimestamp() != nil
	isGenerationUpToDate := physicalMeasurement.Status.UpdateGeneration < physicalMeasurement.ObjectMeta.Generation

	if isDeleted {
		switch physicalMeasurement.Status.State {
		case StateDelete:
			break
		default:
			physicalMeasurement.Status.State = StateDelete
		}
	}

	if isGenerationUpToDate && !isDeleted {
		println("We go to state running...")
		physicalMeasurement.Status.State = StateRunning
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
