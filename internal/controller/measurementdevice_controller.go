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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
)

// MeasurementDeviceReconciler reconciles a MeasurementDevice object
type MeasurementDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func determineState(ctx context.Context, req ctrl.Request) string {
	//TODO
	return ""
}

func executeActions(actions []int, ctx context.Context, req ctrl.Request) []chantico.MeasurementDevice {
	//TODO
	return []chantico.MeasurementDevice{}
}

func (r *MeasurementDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// state := determineState(ctx, req)
	// actions, ok := chantico.ActionMap[state]
	// if !ok {
	// 	return ctrl.Result{}, nil
	// }
	// executeActions(actions, ctx, req)
	return ctrl.Result{}, nil
}
