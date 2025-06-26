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
)

const (
	StatePending   = "Pending"
	StateRunning   = "Running"
	StateCompleted = "Completed"
	StateFailed    = "Failed"
	StateReloaded  = "Reloaded"
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
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	fmt.Printf("\n\n==Measurement: %s==\n", measurementDevice.GetName())
	fmt.Printf("STATE: %s\n", measurementDevice.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(measurementDevice.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

	if measurementDevice.Status.Generation < measurementDevice.ObjectMeta.Generation {
		measurementDevice.Status.State = ""
	}

	switch measurementDevice.Status.State {
	case "":
		return r.startJob(ctx, measurementDevice, req)
	case StateRunning:
		return r.checkJobStatus(ctx, measurementDevice, req)
	case StateCompleted:
		return r.reloadDeployment(ctx, measurementDevice, req)
	case StateFailed:
		return ctrl.Result{}, nil
	case StateReloaded:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown state: %s", measurementDevice.Status.State)
	}
}

func (r *MeasurementDeviceReconciler) startJob(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, req ctrl.Request) (ctrl.Result, error) {
	measurementDevices := &chanticov1alpha1.MeasurementDeviceList{}
	err := r.List(ctx, measurementDevices)
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
			measurementDevice.Status.State = StateFailed
			measurementDevice.Status.ErrorMessage = err.Error()
			_ = r.Status().Update(ctx, measurementDevice)
			return ctrl.Result{}, err
		}
	}

	measurementDevice.Status.State = StateRunning
	measurementDevice.Status.JobName = jobName
	measurementDevice.Status.JobStatus = "Running"
	measurementDevice.Status.LastUpdated = time.Now().Format(time.RFC3339)
	measurementDevice.Status.Generation = measurementDevice.ObjectMeta.Generation
	err = r.Status().Update(ctx, measurementDevice)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *MeasurementDeviceReconciler) checkJobStatus(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, req ctrl.Request) (ctrl.Result, error) {
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{Name: measurementDevice.Status.JobName, Namespace: req.Namespace}, job)
	if err != nil {
		measurementDevice.Status.State = StateFailed
		measurementDevice.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, measurementDevice)
		return ctrl.Result{}, err
	}

	if r.isJobComplete(job) {
		measurementDevice.Status.State = StateCompleted
		measurementDevice.Status.JobStatus = "Completed"
		measurementDevice.Status.LastUpdated = time.Now().Format(time.RFC3339)
		_ = r.Status().Update(ctx, measurementDevice)
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *MeasurementDeviceReconciler) reloadDeployment(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, _ ctrl.Request) (ctrl.Result, error) {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: "chantico-snmp", Namespace: "chantico"}, deployment)
	if err != nil {
		measurementDevice.Status.State = StateFailed
		measurementDevice.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, measurementDevice)
		return ctrl.Result{}, err
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations["reloadedAt"] = time.Now().Format(time.RFC3339)
	err = r.Update(ctx, deployment)
	if err != nil {
		measurementDevice.Status.State = StateFailed
		measurementDevice.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, measurementDevice)
		return ctrl.Result{}, err
	}

	measurementDevice.Status.State = StateReloaded
	measurementDevice.Status.JobStatus = "Reloaded"
	measurementDevice.Status.LastUpdated = time.Now().Format(time.RFC3339)
	_ = r.Status().Update(ctx, measurementDevice)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MeasurementDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.MeasurementDevice{}).
		Complete(r)
}
