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

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	md "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/internal/measurementdevice"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MeasurementDeviceReconciler reconciles a MeasurementDevice object
type MeasurementDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *MeasurementDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the information needed to determine the state of the MeasurementDevice
	measurementDevice := &chantico.MeasurementDevice{}
	_ = r.Get(ctx, req.NamespacedName, measurementDevice)

	measurementDevices := &chantico.MeasurementDeviceList{}
	_ = r.List(ctx, measurementDevices)

	job := &batchv1.Job{}
	_ = r.Get(ctx, client.ObjectKey{Name: measurementDevice.Status.JobName, Namespace: "chantico"}, job)

	deployment := &appsv1.Deployment{}
	_ = r.Get(ctx, client.ObjectKey{Name: "chantico-snmp", Namespace: "chantico"}, deployment)

	state := md.GetState(measurementDevice, measurementDevices.Items, job, deployment)
	md.ExecuteActions(state, ctx, req, measurementDevice, measurementDevices.Items)
	return ctrl.Result{}, nil
}
