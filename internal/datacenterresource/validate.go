package datacenterresource

import (
	"fmt"

	chantico "chantico/api/v1alpha1"
)

const (
	DataCenterResourceTypePDU        = "pdu"
	DataCenterResourceTypeBaremetal  = "baremetal"
	DataCenterResourceTypeVM         = "vm"
	DataCenterResourceTypeKubernetes = "kubernetes"
	DataCenterResourceTypeHeat       = "heat"
)

type ErrorResourceNotFound struct {
	Name string
}

func (e ErrorResourceNotFound) Error() string {
	return fmt.Sprintf("could not locate resource: %s", e.Name)
}

type ErrorCycleDetected struct {
}

func (e ErrorCycleDetected) Error() string {
	return "cyclic loop detected in data center resources"
}

type ErrorUnknownType struct {
	Type string
}

func (e ErrorUnknownType) Error() string {
	return fmt.Sprintf("unknown type: %s", e.Type)
}

func Validate(
	datacenterResource *chantico.DataCenterResource,
	datacenterResources []chantico.DataCenterResource,
	physicalMeasurements []chantico.PhysicalMeasurement,
) ([]string, error) {
	// Perform validation of parent for directed acyclic graph
	resourcesMap := make(map[string]chantico.DataCenterResource)
	for _, resource := range datacenterResources {
		resourcesMap[resource.ObjectMeta.Name] = resource
	}
	queue := make([]string, 0)
	queue = append(queue, datacenterResource.Spec.Parent...)
	visited := 0
	for len(queue) > 0 {
		current, ok := resourcesMap[queue[visited]]
		if !ok {
			return queue[0:visited], ErrorResourceNotFound{Name: queue[visited]}
		}
		if current.ObjectMeta.Name == datacenterResource.ObjectMeta.Name {
			return queue[0:visited], ErrorCycleDetected{}
		}
		visited = visited + 1
		queue = append(queue, current.Spec.Parent...)
	}

	// Check if physical measurements exist
	// TODO(user): For now this validation is skipped because we do not know which
	// order the resources are created

	// Check type of resource
	switch datacenterResource.Spec.Type {
	case "", DataCenterResourceTypePDU, DataCenterResourceTypeBaremetal, DataCenterResourceTypeVM, DataCenterResourceTypeKubernetes, DataCenterResourceTypeHeat:
		return queue[:visited], nil
	default:
		return queue[:visited], ErrorUnknownType{Type: datacenterResource.Spec.Type}
	}
}
