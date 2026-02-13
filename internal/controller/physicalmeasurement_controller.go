/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"log"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chantico "chantico/api/v1alpha1"
	ph "chantico/internal/patch"
	pm "chantico/internal/physicalmeasurement"
)

// PhysicalMeasurementReconciler reconciles a PhysicalMeasurement object
type PhysicalMeasurementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// const (
// 	PhysicalMeasurementStateRunning   = "Running"
// 	PhysicalMeasurementStateCompleted = "Completed"
// 	PhysicalMeasurementStateFailed    = "Failed"
// 	PhysicalMeasurementStateReloaded  = "Reloaded"
// )

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=physicalmeasurements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=physicalmeasurements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=physicalmeasurements/finalizers,verbs=create;update;patch

func (r *PhysicalMeasurementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	physicalMeasurement := &chantico.PhysicalMeasurement{}
	err := r.Get(ctx, req.NamespacedName, physicalMeasurement)
	if err != nil {
		return ctrl.Result{}, nil
	}

	job := &batchv1.Job{}
	_ = r.Get(ctx, client.ObjectKey{Name: physicalMeasurement.Status.JobName, Namespace: "chantico"}, job)

	log.Printf("Updating state of physical measurement %s\n", physicalMeasurement.Name)
	patch := ph.Initialize(ctx, r.Client, physicalMeasurement)
	pm.UpdateState(physicalMeasurement, job)
	patch.PatchStatus()

	result := pm.ExecuteActions(ctx, r.Client, physicalMeasurement, patch)
	if result != nil && (result.Requeue || result.RequeueAfter > 0) {
		return *result.Result, nil
	}
	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *PhysicalMeasurementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chantico.PhysicalMeasurement{}).
		Complete(r)
}
