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
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	chantico "chantico/api/v1alpha1"
)

const (
	DataCenterResourceTypePDU = "pdu"
	DataCenterResourceTypeBaremetal = "baremetal"
	DataCenterResourceTypeVM = "vm"
	DataCenterResourceTypeKubernetes = "kubernetes"
	DataCenterResourceTypeHeat = "heat"
)

// DataCenterResourceReconciler reconciles a DataCenterResource object
type DataCenterResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=datacenterresources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=datacenterresources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=datacenterresources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *DataCenterResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	datacenterResource := &chantico.DataCenterResource{}
	_ = r.Get(ctx, req.NamespacedName, datacenterResource)

	physicalMeasurements := &chantico.PhysicalMeasurementList{}
	_ = r.List(ctx, physicalMeasurements)

	//parent := &chantico.DataCenterResourceList{}
	//_ = r.List(ctx, parent client.ObjectKey{Name: datacenterResource.Spec.Parent, Namespace: "chantico"}, parent)

	// TODO(user): do something with the types/links here:
	// perform operations to make the cluster state reflect the state specified by
	// the user.
	// Specifically: register in postgres (or prometheus?) which datacenter resource
	// is involved for which physical measurement
	// Also perform validation of parent for directed acyclic graph

	switch datacenterResource.Spec.Type {
	case "":
		return ctrl.Result{}, nil
	case DataCenterResourceTypePDU:
		return ctrl.Result{}, nil
	case DataCenterResourceTypeBaremetal:
		return ctrl.Result{}, nil
	case DataCenterResourceTypeVM:
		return ctrl.Result{}, nil
	case DataCenterResourceTypeKubernetes:
		return ctrl.Result{}, nil
	case DataCenterResourceTypeHeat:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown type: %s", datacenterResource.Spec.Type)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataCenterResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chantico.DataCenterResource{}).
		Named("datacenterresource").
		Complete(r)
}
