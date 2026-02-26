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

	// "log"

	chantico "chantico/api/v1alpha1"
	md "chantico/internal/measurementdevice"

	batchv1 "k8s.io/api/batch/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ph "chantico/internal/patch"
)

// MeasurementDeviceReconciler reconciles a MeasurementDevice object
type MeasurementDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices/finalizers,verbs=create;update;patch

func (r *MeasurementDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the information needed to determine the state of the MeasurementDevice
	measurementDevice := &chantico.MeasurementDevice{}
	err := r.Get(ctx, req.NamespacedName, measurementDevice)
	if err != nil {
		return ctrl.Result{}, nil
	}

	job := &batchv1.Job{}
	_ = r.Get(ctx, client.ObjectKey{Name: measurementDevice.Status.JobName, Namespace: "chantico"}, job)

	log.Printf("Updating state of measurement device %s\n", measurementDevice.Name)
	patch := ph.Initialize(ctx, r.Client, measurementDevice)
	md.UpdateState(measurementDevice, job)
	patch.PatchStatus()

	result := md.ExecuteActions(ctx, r.Client, measurementDevice, patch)
	if result != nil && result.Result != nil {
		return *result.Result, nil
	}
	return ctrl.Result{}, nil
}

func (r *MeasurementDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chantico.MeasurementDevice{}).
		Complete(r)
}
