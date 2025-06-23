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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "k8s.io/api/batch/v1"
	appsv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chanticov1alpha1 "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MeasurementReconciler reconciles a Measurement object
type MeasurementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurements/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Measurement object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *MeasurementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	measurement := &chanticov1alpha1.Measurement{}
	err := r.Get(ctx, req.NamespacedName, measurement)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return reconcile.Result{}, err
		}
		aggregator_cron := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "aggregator-" + req.Name,
				Namespace: req.Namespace,
			},
		}
		err = r.Delete(ctx, aggregator_cron)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	measurement.Register(ctx)

	var backOffLimit int32 = 1
	aggregator_cron := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aggregator-" + req.Name,
			Namespace: req.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit: &backOffLimit,
					Template: corev1.PodTemplateSpec{
						Spec: appsv1.PodSpec{
							ImagePullSecrets: []corev1.LocalObjectReference{
								{
									Name: "chantico-gitlab-pull",
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "get-data",
									Image: "ci.tno.nl:4567/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico-aggregator:0.0.2",
									Command: []string{
										"/bin/sh",
									},
									Args: []string{
										"-c",
										fmt.Sprintf(
											"chantico-aggregator -uuid %s -pgdbstring postgres://ps_user:SecurePassword@${POSTGRES_SERVICE_HOST}:${POSTGRES_SERVICE_PORT}/ps_db",
											measurement.UID,
										),
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "chantico-pvc",
											MountPath: "/tmp/snmp-prometheus-volume-mount/",
											SubPath:   "aggregation-volume",
										},
									},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
							Volumes: []corev1.Volume{
								{
									Name: "chantico-pvc",
									VolumeSource: corev1.VolumeSource{
										PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
											ClaimName: "chantico-snmp-prometheus-volume-claim",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set measurement instance as the owner and controller
	if err := controllerutil.SetControllerReference(measurement, aggregator_cron, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Create or update the cron job
	err = r.Create(ctx, aggregator_cron)
	if err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			return reconcile.Result{}, err
		}
	}

	err = r.Update(ctx, aggregator_cron)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MeasurementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.Measurement{}).
		Complete(r)
}
