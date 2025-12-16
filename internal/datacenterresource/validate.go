package datacenterresource

import (
	"fmt"
	"slices"

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
	InvolvedResource string
}

func (e ErrorResourceNotFound) Error() string {
	return fmt.Sprintf("could not locate resource: %s", e.InvolvedResource)
}

type ErrorCycleDetected struct {
	InvolvedResource string
}

func (e ErrorCycleDetected) Error() string {
	return fmt.Sprintf("cyclic loop detected in data center resources from child %s", e.InvolvedResource)
}

type ErrorUnknownType struct {
	Type string
}

func (e ErrorUnknownType) Error() string {
	return fmt.Sprintf("unknown type: %s", e.Type)
}

func GetFromMap(
	resourcesMap map[string]chantico.DataCenterResource,
	nodes []string,
) []chantico.DataCenterResource {
	result := make([]chantico.DataCenterResource, len(nodes))
	for index, node := range nodes {
		result[index] = resourcesMap[node]
	}
	return result
}

func Validate(
	datacenterResource *chantico.DataCenterResource,
	datacenterResources []chantico.DataCenterResource,
	physicalMeasurements []chantico.PhysicalMeasurement,
) ([]chantico.DataCenterResource, error, string) {
	// Perform validation of parent for directed acyclic graph
	resourcesMap := make(map[string]chantico.DataCenterResource)
	for _, resource := range datacenterResources {
		if resource.Status.State != StateDelete {
			resourcesMap[resource.ObjectMeta.Name] = resource
		}
	}
	queue := make([]string, 0)
	queue = append(queue, datacenterResource.Spec.Parent...)
	visited := 0
	for len(queue) > visited {
		current, ok := resourcesMap[queue[visited]]
		if !ok {
			return GetFromMap(resourcesMap, queue[0:visited]), ErrorResourceNotFound{InvolvedResource: queue[visited]}, queue[visited]
		}
		if slices.Contains(current.Spec.Parent, datacenterResource.ObjectMeta.Name) {
			return GetFromMap(resourcesMap, queue[0:visited]), ErrorCycleDetected{InvolvedResource: queue[visited]}, queue[visited]
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
		return GetFromMap(resourcesMap, queue[0:visited]), nil, ""
	default:
		return GetFromMap(resourcesMap, queue[0:visited]), ErrorUnknownType{Type: datacenterResource.Spec.Type}, ""
	}
}
