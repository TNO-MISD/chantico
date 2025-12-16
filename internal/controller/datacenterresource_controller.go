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

	"k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	chantico "chantico/api/v1alpha1"
	dcr "chantico/internal/datacenterresource"
	ph "chantico/internal/patch"
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

	listOptions := []client.ListOption{client.InNamespace(req.NamespacedName.Namespace)}
	datacenterResources := &chantico.DataCenterResourceList{}
	_ = r.List(ctx, datacenterResources, listOptions...)

	physicalMeasurements := &chantico.PhysicalMeasurementList{}
	_ = r.List(ctx, physicalMeasurements, listOptions...)

	patch := ph.Initialize(ctx, r.Client, datacenterResource)

	// Update state of the resource
	log.Printf("Updating state of data center resource %s\n", datacenterResource.Name)
	dcr.UpdateState(datacenterResource)
	patch.PatchStatus()

	log.Printf("Object post-update status: %#v\n", datacenterResource.Status.State)
	result := dcr.ExecuteActions(ctx, r.Client, datacenterResource)
	log.Printf("Finished executing actions\n")
	if result != nil {
		if result.Requeue || result.RequeueAfter > 0 {
			return result.Result, nil
		}
		if result.UpdateSpec {
			log.Printf("Patch spec\n")
			patch.PatchSpec()
		}
		if result.UpdateStatus {
			log.Printf("Patch status\n")
			patch.PatchStatus()
		}
	}

	// Perform validation and clear other visited node validation errors if needed
	// This brings those into a reconciliation loop as well
	visited, err, involvedResource := dcr.Validate(datacenterResource, datacenterResources.Items, physicalMeasurements.Items)
	if err != nil {
		log.Printf("Setting validation error of data center resource %s: %s\n", datacenterResource.Name, err)
		dcr.SetValidationError(datacenterResource, err, involvedResource)
	} else {
		log.Printf("Clearing validation errors of data center resource %s", datacenterResource.Name)
		log.Printf("Previous status: %#v", datacenterResource.Status)
		for _, node := range visited {
			log.Printf("Checking visited node %s\n", node)
			r.ClearReferencedValidation(ctx, req, node)
		}
		references := &chantico.DataCenterResourceList{}
		_ = r.List(ctx, references, append(listOptions, client.MatchingFields{"status.involvedResource": datacenterResource.Name})...)
		for _, reference := range references.Items {
			log.Printf("Checking referenced node %s\n", reference.Name)
			r.ClearReferencedValidation(ctx, req, reference.Name)
		}
		if datacenterResource.Status.InvolvedResource != "" {
			log.Printf("Checking involved resource %s\n", datacenterResource.Status.InvolvedResource)
			r.ClearReferencedValidation(ctx, req, datacenterResource.Status.InvolvedResource)
		}
		dcr.ClearValidationError(datacenterResource)
	}
	patch.PatchStatus()

	// TODO(user): do something with the links here:
	// perform operations to make the cluster state reflect the state specified by
	// the user.
	// Specifically: register in postgres (or prometheus?) which datacenter resource
	// is involved for which physical measurement

	return ctrl.Result{}, nil
}

func (r *DataCenterResourceReconciler) ClearReferencedValidation(
	ctx context.Context,
	req ctrl.Request,
	node string,
) {
	resource := &chantico.DataCenterResource{}
	_ = r.Get(ctx, types.NamespacedName{Namespace: req.NamespacedName.Namespace, Name: node}, resource)
	patch := ph.Initialize(ctx, r.Client, resource)
	dcr.ClearValidationError(resource)
	patch.PatchStatus()
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataCenterResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chantico.DataCenterResource{}).
		Named("datacenterresource").
		Complete(r)
}
