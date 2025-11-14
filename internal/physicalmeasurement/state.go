package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
	"fmt"
	"strconv"
)

const (
	StateInit      = "init"
	StateRunning   = "Running"
	StateCompleted = "Completed"
	StateFailed    = "Failed"
	StateReloaded  = "Reloaded"
)

func GetState(
	physicalMeasurement *chantico.PhysicalMeasurement,
	// job *batchv1.Job,
	// deployment *appsv1.Deployment,
) string {
	if physicalMeasurement == nil {
		return StateFailed
	}

	fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
	fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

	if physicalMeasurement.Status.Generation < physicalMeasurement.ObjectMeta.Generation {
		physicalMeasurement.Status.State = ""
	}

	switch physicalMeasurement.Status.State {
	case "":
		return StateInit
	default:
		panic("Not implemented yet")
	}
}
