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

func Validate(
	datacenterResource *chantico.DataCenterResource,
	datacenterResources []chantico.DataCenterResource,
	physicalMeasurements []chantico.PhysicalMeasurement,
) error {
	// Perform validation of parent for directed acyclic graph
	resourcesMap := make(map[string]chantico.DataCenterResource)
	for _, resource := range datacenterResources {
		resourcesMap[resource.ObjectMeta.Name] = resource
	}
	queue := make([]string, 0)
	queue = append(queue, datacenterResource.Spec.Parent...)
	for len(queue) > 0 {
		current, ok := resourcesMap[queue[0]]
		if !ok {
			return ErrorResourceNotFound{Name: queue[0]}
		}
		if current.ObjectMeta.Name == datacenterResource.ObjectMeta.Name {
			return ErrorCycleDetected{}
		}
		queue = queue[1:]
		queue = append(queue, current.Spec.Parent...)
	}

	// Check if physical measurements exist
	// TODO(user): For now this validation is skipped because we do not know which
	// order the resources are created

	// Check type of resource
	switch datacenterResource.Spec.Type {
	case "":
		return nil
	case DataCenterResourceTypePDU:
		return nil
	case DataCenterResourceTypeBaremetal:
		return nil
	case DataCenterResourceTypeVM:
		return nil
	case DataCenterResourceTypeKubernetes:
		return nil
	case DataCenterResourceTypeHeat:
		return nil
	default:
		return fmt.Errorf("unknown type: %s", datacenterResource.Spec.Type)
	}
}
