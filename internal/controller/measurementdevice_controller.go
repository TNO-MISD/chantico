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
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	chanticov1alpha1 "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MeasurementDeviceReconciler reconciles a MeasurementDevice object
type MeasurementDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *MeasurementDeviceReconciler) isJobComplete(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *MeasurementDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	measurementDevice := &chanticov1alpha1.MeasurementDevice{}
	err := r.Get(ctx, req.NamespacedName, measurementDevice)
	if err != nil {
		return ctrl.Result{}, err
	}

	measurementDevices := &chanticov1alpha1.MeasurementDeviceList{}
	err = r.List(ctx, measurementDevices)

	if err != nil {
		return ctrl.Result{}, err
	}

	uidString := string(measurementDevice.UID)
	jobName := fmt.Sprintf("update-snmp-%s-%s", uidString, strconv.FormatInt(time.Now().UnixMilli(), 10))
	fmt.Printf("%s\n", jobName)

	var configLines []string
	configLines = append(configLines, "auths:")
	configLines = append(configLines, "  public_v3:")
	configLines = append(configLines, "    version: 3")
	configLines = append(configLines, "    username: guest")
	configLines = append(configLines, "modules:")

	for _, device := range measurementDevices.Items {
		configLines = append(configLines, fmt.Sprintf("  %s:", device.GetName()))
		configLines = append(configLines, fmt.Sprintf("    walk: [%s]", strings.Join(device.Spec.Walks, ",")))
	}

	filePath := fmt.Sprintf("/tmp/chantico-volume-mount/snmp/config/generator-%s.yml", uidString)
	var configCommands []string
	configCommands = append(configCommands, fmt.Sprintf("rm -f %s", filePath))
	configCommands = append(configCommands, fmt.Sprintf("touch %s", filePath))
	for _, configLine := range configLines {
		configCommands = append(configCommands, fmt.Sprintf("echo '%s' >> %s", configLine, filePath))
	}
	configCommand := strings.Join(configCommands, "\n")

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: req.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "create-snmp-generator",
							Image: "busybox",
							Command: []string{
								"sh",
								"-c",
								configCommand,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "chantico-volume-mount",
									MountPath: "/tmp/chantico-volume-mount/snmp/",
									SubPath:   "snmp/",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "create-snmp-config",
							Image: "prom/snmp-generator",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "chantico-volume-mount",
									MountPath: "/opt/snmp.yml",
									SubPath:   "snmp/yml/snmp.yml",
								},
								{
									Name:      "chantico-volume-mount",
									MountPath: "/opt/generator.yml",
									SubPath:   fmt.Sprintf("snmp/config/generator-%s.yml", uidString),
								},
								{
									Name:      "chantico-volume-mount",
									MountPath: "/opt/mibs",
									SubPath:   "snmp/mibs",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: "chantico-volume-mount",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "volume-test",
								},
							},
						},
					},
				},
			},
		},
	}
	err = r.Create(ctx, job)

	if err != nil {
		if client.IgnoreAlreadyExists(err) != nil {
			return reconcile.Result{}, err
		}
	}

	go func() {
		// Polling mechanism to wait for job completion
		for {
			err = r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: req.Namespace}, job)
			if err != nil {
				return
			}

			if r.isJobComplete(job) {
				break
			}

			time.Sleep(time.Second) // Poll every 10 seconds
		}

		// Reload the deployment after the job is complete
		deployment := &appsv1.Deployment{}
		err = r.Get(ctx, client.ObjectKey{Name: "chantico-snmp", Namespace: "chantico"}, deployment)
		if err != nil {
			return
		}

		// Update the deployment to trigger a reload
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = map[string]string{}
		}
		deployment.Spec.Template.Annotations["reloadedAt"] = time.Now().Format(time.RFC3339)
		err = r.Update(ctx, deployment)
		if err != nil {
			return
		}
	}()

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MeasurementDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.MeasurementDevice{}).
		Complete(r)
}
