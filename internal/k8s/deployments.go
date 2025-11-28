package k8s

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
)

func CheckDeploymentAvailability(deployment appsv1.Deployment, gracePeriod time.Duration) bool {
	if deployment.ObjectMeta.Generation != deployment.Status.ObservedGeneration {
		return false
	}
	if deployment.Status.UnavailableReplicas != 0 {
		return false
	}
	return true
}
