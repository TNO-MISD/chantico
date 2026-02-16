package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
)

func CheckDeploymentAvailability(deployment appsv1.Deployment) bool {
	if deployment.ObjectMeta.Generation != deployment.Status.ObservedGeneration {
		return false
	}
	if deployment.Status.UnavailableReplicas != 0 {
		return false
	}
	return true
}
