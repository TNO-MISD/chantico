package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
	"fmt"
	"strconv"
)

const (
	StateInit      = "init"
	StateRunning   = "Running"
	StateDeleted   = "Deleted"
	StateFailed    = "Failed"
	StateCompleted = "Completed"
)

func GetState(
	physicalMeasurement *chantico.PhysicalMeasurement,
) string {
	if physicalMeasurement == nil {
		return StateCompleted
	}

	fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
	fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

	switch physicalMeasurement.Status.State {
	case "":
		return StateInit
	default:
		panic("Not implemented yet")
	}
}
