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
	"slices"
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

	"k8s.io/client-go/util/retry"

	chanticov1alpha1 "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
)

const (
	MeasurementDeviceStateStart     = "Start"
	MeasurementDeviceStateUpdating  = "Updating"
	MeasurementDeviceStateUpdated   = "Updated"
	MeasurementDeviceStateReloading = "Reloading"
	MeasurementDeviceStateReloaded  = "Reloaded"
	MeasurementDeviceStateFailed    = "Failed"

	MeasurementDeviceStateDeleting = "Deleting"
	MeasurementDeviceStateDeleted  = "Deleted"
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

const measurementDeviceFinalizer = "measurementDevice.finalizer.chantico.ci.tno.nl"

func (r *MeasurementDeviceReconciler) setFinalizer(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice) (ctrl.Result, error) {
	if measurementDevice.ObjectMeta.Finalizers == nil {
		measurementDevice.ObjectMeta.Finalizers = []string{}
	}

	measurementDevice.ObjectMeta.Finalizers = append(measurementDevice.ObjectMeta.Finalizers, measurementDeviceFinalizer)
	return ctrl.Result{Requeue: true}, r.Update(ctx, measurementDevice)
}

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevice,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevice/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevice/finalizers,verbs=update

func (r *MeasurementDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// Intialize measurementDevice
	measurementDevice := &chanticov1alpha1.MeasurementDevice{}
	err := r.Get(ctx, req.NamespacedName, measurementDevice)

	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Printf("Reconciliation loop started for %v at %v\n", measurementDevice.UID, req.NamespacedName, time.Now().Format(time.RFC3339))

	// Set finalizer if needed
	if !slices.Contains(measurementDevice.ObjectMeta.Finalizers, measurementDeviceFinalizer) {
		return r.setFinalizer(ctx, measurementDevice)
	}

	// Check if new
	if measurementDevice.Status.State == "" || measurementDevice.Status.State != MeasurementDeviceStateStart && measurementDevice.Status.Generation < measurementDevice.ObjectMeta.Generation {
		measurementDevice.Status.State = MeasurementDeviceStateStart
		measurementDevice.Status.Generation = measurementDevice.ObjectMeta.Generation
		return ctrl.Result{Requeue: true}, r.Status().Update(ctx, measurementDevice)
	}

	// Check if is
	if measurementDevice.ObjectMeta.DeletionTimestamp.IsZero() {
		switch measurementDevice.Status.State {
		case MeasurementDeviceStateStart:
			return r.updateSnmp(ctx, measurementDevice, req)
		case MeasurementDeviceStateUpdating:
			return r.monitorUpdateSnmp(ctx, measurementDevice, req)
		case MeasurementDeviceStateUpdated:
			return r.reloadSnmp(ctx, measurementDevice, req)
		case MeasurementDeviceStateReloading:
			return r.monitorReloadSnmp(ctx, measurementDevice, req)
		case MeasurementDeviceStateReloaded:
			return ctrl.Result{}, nil
		case MeasurementDeviceStateFailed:
			return ctrl.Result{}, nil
		default:
			return ctrl.Result{}, fmt.Errorf("%s: Should not reach that state", measurementDevice.UID)
		}

	} else {
		measurementDevice.ObjectMeta.Finalizers = []string{}
		return ctrl.Result{}, r.Update(ctx, measurementDevice)
		// switch measurementDevice.Status.State {
		// case MeasurementDeviceStateStart:
		// 	return r.checkJobStatus(ctx, measurementDevice, req)
		// case MeasurementDeviceStateCompleted:
		// 	return r.reloadDeployment(ctx, measurementDevice, req)
		// case MeasurementDeviceStateFailed:
		// 	return ctrl.Result{}, nil
		// case MeasurementDeviceStateReloaded:
		// 	return ctrl.Result{}, nil
		// default:
		// 	return ctrl.Result{}, fmt.Errorf("%s: Should not reach that state", measurementDevice.UID)
		// }
	}
}

func (r *MeasurementDeviceReconciler) updateSnmp(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, req ctrl.Request) (ctrl.Result, error) {
	measurementDevices := &chanticov1alpha1.MeasurementDeviceList{}
	err := r.List(ctx, measurementDevices)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Determine if update is currently possible
	var configLines []string

	toUpdateDevices := []chanticov1alpha1.MeasurementDevice{}
	isCurrentlyUpdating := false
	for _, device := range measurementDevices.Items {
		fmt.Printf("\nREF: %#v: %#v\n", measurementDevice.UID, measurementDevice.Status.LastUpdated)
		fmt.Printf("DEVICE: %#v: %#v\n\n", device.UID, device.Status.LastUpdated)
		if device.Status.LastUpdated == "" || device.Status.Generation < device.ObjectMeta.Generation {
			toUpdateDevices = append(toUpdateDevices, device)
		} else {
		}
		isCurrentlyUpdating = isCurrentlyUpdating || device.Status.State == MeasurementDeviceStateUpdating || device.Status.State == MeasurementDeviceStateUpdated || device.Status.State == MeasurementDeviceStateReloading
	}

	if isCurrentlyUpdating {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Create new configuration
	updateTimestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	jobName := fmt.Sprintf("update-snmp-%s", updateTimestamp)
	fmt.Printf("%s\n", jobName)

	configLines = append(configLines, "auths:")
	configLines = append(configLines, "  public_v3:")
	configLines = append(configLines, "    version: 3")
	configLines = append(configLines, "    username: guest")
	configLines = append(configLines, "modules:")

	for _, device := range measurementDevices.Items {
		configLines = append(configLines, fmt.Sprintf("  %s:", device.GetName()))
		configLines = append(configLines, fmt.Sprintf("    walk: [%s]", strings.Join(device.Spec.Walks, ",")))
	}

	filePath := fmt.Sprintf("/tmp/chantico-volume-mount/snmp/config/generator-%s.yml", updateTimestamp)
	var configCommands []string
	configCommands = append(configCommands, fmt.Sprintf("rm -f %s", filePath))
	configCommands = append(configCommands, fmt.Sprintf("touch %s", filePath))
	for _, configLine := range configLines {
		configCommands = append(configCommands, fmt.Sprintf("echo '%s' >> %s", configLine, filePath))
	}
	configCommand := strings.Join(configCommands, "\n")

	// Mark the devices as updating
	fmt.Printf("TO UPDATE LENGTH: %d/%d\n", len(toUpdateDevices), len(measurementDevices.Items))
	for _, device := range toUpdateDevices {
		fmt.Printf("%s: updating snmp\n", device.UID)

		device.Status.State = MeasurementDeviceStateUpdating
		device.Status.JobName = jobName
		device.Status.LastUpdated = time.Now().Format(time.RFC3339)
		device.Status.Generation = device.ObjectMeta.Generation
		fmt.Printf("%#v - %#v, %#v\n", device.UID, device.Status.LastUpdated)
		_ = r.Status().Update(ctx, &device)
	}

	// Create snmp configuration update job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "chantico",
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
									SubPath:   fmt.Sprintf("snmp/config/generator-%s.yml", updateTimestamp),
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
			measurementDevice.Status.State = MeasurementDeviceStateFailed
			measurementDevice.Status.ErrorMessage = err.Error()
			_ = r.Status().Update(ctx, measurementDevice)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *MeasurementDeviceReconciler) monitorUpdateSnmp(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, req ctrl.Request) (ctrl.Result, error) {
	// Check the update status
	measurementDevices := &chanticov1alpha1.MeasurementDeviceList{}
	err := r.List(ctx, measurementDevices)
	if err != nil {
		return ctrl.Result{}, err
	}

	job := &batchv1.Job{}
	for _, device := range measurementDevices.Items {
		_ = r.Get(ctx, client.ObjectKey{Name: device.Status.JobName, Namespace: "chantico"}, job)
		if job.Status.CompletionTime == nil || device.Status.State == MeasurementDeviceStateReloading {
			fmt.Printf("%s: update job is not finished\n", measurementDevice.UID)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	fmt.Printf("%s: updated snmp\n", measurementDevice.UID)
	measurementDevice.Status.State = MeasurementDeviceStateUpdated
	return ctrl.Result{}, r.Status().Update(ctx, measurementDevice)
}

func (r *MeasurementDeviceReconciler) reloadSnmp(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, req ctrl.Request) (ctrl.Result, error) {
	fmt.Printf("%s: reload snmp\n", measurementDevice.UID)

	err := retry.RetryOnConflict(
		retry.DefaultRetry,
		func() error {
			deployment := &appsv1.Deployment{}
			err := r.Get(ctx, client.ObjectKey{Name: "chantico-snmp", Namespace: "chantico"}, deployment)

			if err != nil {
				return err
			}

			// Reload deployment
			if deployment.Spec.Template.Annotations == nil {
				deployment.Spec.Template.Annotations = map[string]string{}
			}
			deployment.Spec.Template.Annotations["reloadedAt"] = time.Now().Format(time.RFC3339)
			return r.Update(ctx, deployment)
		},
	)

	if err != nil {
		measurementDevice.Status.State = MeasurementDeviceStateFailed
		measurementDevice.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, measurementDevice)
		return ctrl.Result{}, err
	}

	measurementDevice.Status.State = MeasurementDeviceStateReloading
	return ctrl.Result{RequeueAfter: 5 * time.Second}, r.Status().Update(ctx, measurementDevice)
}

func (r *MeasurementDeviceReconciler) monitorReloadSnmp(ctx context.Context, measurementDevice *chanticov1alpha1.MeasurementDevice, req ctrl.Request) (ctrl.Result, error) {

	// Check current deployment
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: "chantico-snmp", Namespace: "chantico"}, deployment)
	if err != nil {
		measurementDevice.Status.State = MeasurementDeviceStateFailed
		measurementDevice.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, measurementDevice)
		return ctrl.Result{}, err
	}

	// Condition
	if deployment.Status.UnavailableReplicas != 0 {
		fmt.Printf("%s: waiting on %s\n", measurementDevice.UID, measurementDevice.Status.JobName)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	var condition appsv1.DeploymentCondition

	for _, condition = range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status != "True" {
			fmt.Printf("%s: waiting on %s\n", measurementDevice.UID, measurementDevice.Status.JobName)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}

	// Make updating measurementDevices reloaded
	measurementDevices := &chanticov1alpha1.MeasurementDeviceList{}
	err = r.List(ctx, measurementDevices)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, device := range measurementDevices.Items {
		if device.Status.State == MeasurementDeviceStateReloading || device.Status.State == MeasurementDeviceStateUpdating {
			fmt.Printf("%s: reloaded snmp\n", measurementDevice.UID)
			device.Status.State = MeasurementDeviceStateReloaded
			_ = r.Status().Update(ctx, &device)
		}
	}
	return ctrl.Result{}, nil
}

func (r *MeasurementDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.MeasurementDevice{}).
		Complete(r)
}
