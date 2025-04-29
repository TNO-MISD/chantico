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

	chanticov1alpha1 "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PduReconciler reconciles a Pdu object
type PduReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func int32Ptr(i int32) *int32 { return &i }

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=pdus,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=pdus/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=pdus/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pdu object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PduReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the PDU instance
	pdu := &chanticov1alpha1.Pdu{}
	err := r.Get(ctx, req.NamespacedName, pdu)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}
		// PDU resource not found, delete the whalesay deployment
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "exporter-" + req.Name,
				Namespace: req.Namespace,
			},
		}
		err = r.Delete(ctx, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// PDU resource found, create the cowsay deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "exporter-" + req.Name,
			Namespace: req.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": fmt.Sprintf("exporter-%s", req.Name),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": fmt.Sprintf("exporter-%s", req.Name),
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "cowsay",
							Image:   "rancher/cowsay",
							Command: []string{"sh", "-c", fmt.Sprintf("while true; do cowsay 'Hello %s:%d'; sleep 10; done", pdu.Spec.Ip, pdu.Spec.Port)},
						},
					},
				},
			},
		},
	}

	// Set PDU instance as the owner and controller
	if err := controllerutil.SetControllerReference(pdu, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update the deployment
	err = r.Create(ctx, deployment)
	if err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			return reconcile.Result{}, err
		}
	}
	err = r.Update(ctx, deployment)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PduReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.Pdu{}).
		Complete(r)
}
