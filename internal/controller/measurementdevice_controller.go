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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strconv"

	// "sigs.k8s.io/cluster-api/util/patch"

	// "io/fs"
	vol "chantico/internal/volumes"
	"log"
	"os"
	"path/filepath"

	chantico "chantico/api/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	// ph "chantico/internal/patch"

	"chantico/internal/snmp"
	"crypto/sha256"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MeasurementDeviceReconciler reconciles a MeasurementDevice object
type MeasurementDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices/finalizers,verbs=create;update;patch

/*
This function is triggered by events. We currently call it MeasurementDevice, but we could rename this to SNMPConfig. We actually provide an interface to the prom/generator.

We can follow the MIB directory convention of the generator. Make it clear that we use SNMP Generator.
kind: SNMPConfig or SNMPGenerator or SNMPConfigGenerator
spec:

	MIBDirectories:
	- ...
	- ...
	generatorConfig:
	...

prometheus (applicatie, container, in docker, of wat dan ook)
prometheus-operator (management van applicatie, operator, alleen in kubernetes)
---
argo workflows (alleen in K8s)

CRD: Chantico
-> Prometheus deployment
-> folders bestaan (snmp/mibs/...)
-> SNMPExporter

CRD: SNMPExporter -> endpoint om SNMP metrics op te halen
CRD: SNMPConfig -> Prom/Generator (MIBS, Generator.yaml) -> snmp.yaml
*/
func (r *MeasurementDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	// Get the measurement device
	measurementDevice := &chantico.MeasurementDevice{}
	err := r.Get(ctx, req.NamespacedName, measurementDevice)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Helper function makes a deep copy of measurement device, and Patches the spec/status as needed at the end of reconcile function.
	helper, err := patch.NewHelper(measurementDevice, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		if err := r.reconcileStatus(ctx, measurementDevice); err != nil {
			reterr = errors.Join(reterr, err)
		}
		if err := helper.Patch(ctx, measurementDevice); err != nil {
			reterr = errors.Join(reterr, err)
		}
	}()

	// Runs all functions, every function performs a check on the actual state, and then decides whether it needs to take action
	functions := []func(context.Context, *chantico.MeasurementDevice) (bool, error){
		r.ensureFinalizerIsSet,
	}
	for _, function := range functions {
		stop, err := function(ctx, measurementDevice)
		if err != nil {
			return ctrl.Result{}, err
		}
		if stop {
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

// maybe some named return for ctrl.Result{}?? so we can simplify the smaller reconcile loops?

// if res := r.reconcileFinalizer(ctx, measurementDevice); res.Stop {
// 	log.Println("stopping finalizer reconcile", res.Error)
// 	return res.Result, res.Error
// }

// if res := r.reconcileGeneratorFile(ctx, measurementDevice); res.Stop {
// 	log.Println("stopping generator file reconcile", res.Error)
// 	return res.Result, res.Error
// }

// if res := r.reconcileSNMPFileExistence(ctx, measurementDevice); res.Stop {
// 	log.Println("stopping SNMP file existence reconcile", res.Error)
// 	return res.Result, res.Error
// }

// // All MIBs are parsed for the walk, how are they unique?

// if res := r.reconcileSNMPGeneratorJob(ctx, measurementDevice); res.Stop {
// 	log.Println("stopping job reconcile", res.Error)
// 	return res.Result, res.Error
// }

// if res := r.reconcileSNMPFile(ctx, measurementDevice); res.Stop {
// 	log.Println("stopping SNMP file reconcile", res.Error)
// 	return res.Result, res.Error

// }

// // look at the new file, add the SHA of snmp-....yaml file so other controllers can see update

// return ctrl.Result{}, nil

type ReconcileResult struct {
	Result ctrl.Result
	Error  error
	Stop   bool
}

/*
Determines the "Ready" condition which is shown to users for a general insight into the status. Currently only depends on "Job" condition, but we can expand this. Or even use conditions of the Cluster API.
*/
func (r *MeasurementDeviceReconciler) reconcileStatus(ctx context.Context, measurementDevice *chantico.MeasurementDevice) error {
	// should use ObservedGeneration for determining up-to-date or old conditions?
	// we should probably also use a global ObservedGeneration (so then we can see what reconcile has been, and whether it matches the conditions)
	jobCondition := meta.FindStatusCondition(measurementDevice.Status.Conditions, "Job")
	if jobCondition == nil {
		meta.SetStatusCondition(&measurementDevice.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionUnknown,
			ObservedGeneration: measurementDevice.Generation,
		})
		return nil
	}

	switch jobCondition.Status {
	case metav1.ConditionFalse:
		meta.SetStatusCondition(&measurementDevice.Status.Conditions, metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionFalse,
		})
	case metav1.ConditionUnknown:
		meta.SetStatusCondition(&measurementDevice.Status.Conditions, metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionUnknown,
		})
	case metav1.ConditionTrue:
		meta.SetStatusCondition(&measurementDevice.Status.Conditions, metav1.Condition{
			Type:   "Ready",
			Status: metav1.ConditionTrue,
		})
	}
	return nil
}

func (r *MeasurementDeviceReconciler) reconcileDeletion() {
	// logic
}

func (r *MeasurementDeviceReconciler) ensureFinalizerIsSet(ctx context.Context, measurementDevice *chantico.MeasurementDevice) (bool, error) {
	if controllerutil.ContainsFinalizer(measurementDevice, chantico.MeasurementDeviceFinalizer) {
		return false, nil
	}

	controllerutil.AddFinalizer(measurementDevice, chantico.MeasurementDeviceFinalizer)
	return true, nil
}

func (r *MeasurementDeviceReconciler) reconcileMibFile(ctx context.Context, measurementDevice *chantico.MeasurementDevice) ReconcileResult {
	/*
		I think we should be more explicit for MIB files, or directories. This way we can prevent name space collisions.
	*/
	return ReconcileResult{}
}

func (r *MeasurementDeviceReconciler) reconcileSNMPFileExistence(ctx context.Context, measurementDevice *chantico.MeasurementDevice) ReconcileResult {
	/*
		We need to have an SNMP file (even if it is empty, it will be filled later by SNMP Generator).
	*/
	// for now create snmp dir, for some reason this is now done from an init container...
	// Chantico CR, then the Chantico controller will create the folders

	pathToFile := filepath.Join(os.Getenv(vol.ChanticoVolumeLocationEnv), "snmp/snmp", fmt.Sprintf("snmp-%s.yaml", measurementDevice.GetUID()))

	_, err := os.ReadFile(pathToFile)
	if err == nil {
		// file exists, awesome
		return ReconcileResult{}
	}
	if !errors.Is(err, fs.ErrNotExist) {
		// another error, maybe permissions, or smth
		return ReconcileResult{Error: err, Stop: true}
	}

	// create file
	dir := filepath.Dir(pathToFile)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return ReconcileResult{Error: err, Stop: true}
	}
	err = os.WriteFile(pathToFile, []byte{}, 0777)
	if err != nil {
		return ReconcileResult{Error: err, Stop: true}
	}
	return ReconcileResult{}
}

func (r *MeasurementDeviceReconciler) reconcileGeneratorFile(ctx context.Context, measurementDevice *chantico.MeasurementDevice) ReconcileResult {
	/*
		get observed generator (from file)
		get desired generator (from spec)
		compare
		update if required

		sidenote: rather than writing to file, you can also update the status
	*/

	pathToFile := filepath.Join(os.Getenv(vol.ChanticoVolumeLocationEnv), "snmp/generators", fmt.Sprintf("generator-%s.yaml", measurementDevice.GetUID()))
	observedGenerator, err := os.ReadFile(pathToFile)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// error when trying to read file, other than not exist error
		return ReconcileResult{Error: err, Stop: true}
	}

	desiredGenerator, err := yaml.Marshal(snmp.GeneratorConfig{
		Auths: map[string]*snmp.GeneratorAuth{
			measurementDevice.Name: &measurementDevice.Spec.Auth,
		},
		Modules: map[string]*snmp.GeneratorModule{
			measurementDevice.Name: {
				Walk: measurementDevice.Spec.Walks,
			},
		},
	})
	if err != nil {
		// maybe add error message to object
		return ReconcileResult{Error: err, Stop: true}
	}

	observedSha := sha256.Sum256(observedGenerator)
	desiredSha := sha256.Sum256(desiredGenerator)
	if bytes.Equal(desiredSha[:], observedSha[:]) {
		// desired == observed, do nothing
		return ReconcileResult{}
	}

	dir := filepath.Dir(pathToFile)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return ReconcileResult{Error: err, Stop: true}
	}

	if err := os.WriteFile(pathToFile, desiredGenerator, 0777); err != nil {
		// error when writing to file
		return ReconcileResult{Error: err, Stop: true}
	}

	patched := measurementDevice.DeepCopy()
	meta.SetStatusCondition(&patched.Status.Conditions, metav1.Condition{
		Type:               "GeneratorFile",
		Status:             metav1.ConditionTrue,
		Reason:             "GeneratorFileGenerated",
		Message:            "Generator file has been generated successfully.",
		ObservedGeneration: measurementDevice.Generation,
	})
	if err := r.Patch(ctx, patched, client.MergeFrom(measurementDevice)); err != nil {
		return ReconcileResult{Error: err, Stop: true}
	}

	// successfully wrote to file
	return ReconcileResult{}
}

func int32Ptr(i int32) *int32 { return &i }

func (r *MeasurementDeviceReconciler) reconcileSNMPGeneratorJob(ctx context.Context, measurementDevice *chantico.MeasurementDevice) ReconcileResult {
	/*
		desired state:
		- there should be a single job
		- with configuration of desired generator file
		- ended succesful
	*/
	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, client.InNamespace(measurementDevice.GetNamespace())); err != nil {
		return ReconcileResult{Error: err, Stop: true}
	}

	// this can be optimized with indexing (at the manager)
	var ownedJobs []batchv1.Job
	for _, job := range jobList.Items {
		for _, ownerRef := range job.OwnerReferences {
			if ownerRef.UID == measurementDevice.GetUID() {
				ownedJobs = append(ownedJobs, job)
			}
		}
	}

	if len(ownedJobs) == 0 {
		// maybe this can be obtained from shared function or from status

		volume, err := vol.GetChanticoVolume() // ugly?
		if err != nil {
			return ReconcileResult{Error: err, Stop: true}
		}

		/*
			mount path - file path within the volume
			so for local development: /tmp/chantico-local-path-data/pvc-e77d4e95-0d5b-4f4b-a390-b625749362da_chantico_chantico-snmp-prometheus-volume-claim + snmp/generators
			for within cluster: /data/snmp/snmp
		*/

		mountPath := "/data"

		generatorPath := filepath.Join(mountPath, "snmp/generators", fmt.Sprintf("generator-%s.yaml", measurementDevice.GetUID()))
		mibsDir := filepath.Join(mountPath, "snmp/mibs")
		outputPath := filepath.Join(mountPath, "snmp/snmp", fmt.Sprintf("snmp-%s.yaml", measurementDevice.GetUID()))

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      measurementDevice.GetName(),
				Namespace: measurementDevice.GetNamespace(),
				Annotations: map[string]string{
					"measurementdevice.generation.chantico": strconv.FormatInt(measurementDevice.GetGeneration(), 10),
				},
			},
			Spec: batchv1.JobSpec{

				BackoffLimit: int32Ptr(0),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "snmp-generator",
								Image: "prom/snmp-generator:v0.29.0",
								Command: []string{
									"/bin/generator",
								},
								Args: []string{
									"generate",
									"--output-path", outputPath,
									"--generator-path", generatorPath,
									"--mibs-dir", mibsDir,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      vol.ChanticoVolumeMount,
										MountPath: mountPath,
									},
								},
							},
						},
						Volumes:       []corev1.Volume{volume},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(measurementDevice, job, r.Scheme); err != nil {
			return ReconcileResult{Error: err, Stop: true}
		}

		if err := r.Create(ctx, job); err != nil {
			return ReconcileResult{Error: err, Stop: true}
		}

		log.Println("creating job")
		return ReconcileResult{Stop: true}

	} else if len(ownedJobs) == 1 {
		job := ownedJobs[0]

		annotations := job.GetAnnotations()

		observedGeneration, exists := annotations["measurementdevice.generation.chantico"]
		if !exists {
			err := fmt.Errorf("Annotation has not been set for job. Should not be possible.")
			return ReconcileResult{Error: err, Stop: true}
		}
		desiredGeneration := strconv.FormatInt(measurementDevice.GetGeneration(), 10)
		if observedGeneration != desiredGeneration {
			// job is not up to date
			if err := r.Delete(ctx, &job); err != nil {
				err := fmt.Errorf("Could not delete job.")
				return ReconcileResult{Error: err, Stop: true}
			}
			return ReconcileResult{Stop: true}
		}

		if !isJobSuccessful(&job) {
			// this is actually not correct, we should check if job failed, or if it is still pending
			// patched := measurementDevice.DeepCopy()
			// meta.SetStatusCondition(&patched.Status.Conditions, metav1.Condition{
			// 	Type:               "JobSucceeded",
			// 	Status:             metav1.ConditionUnknown,
			// 	Reason:             "JobPending",
			// 	ObservedGeneration: measurementDevice.Generation,
			// })
			// if err := r.Patch(ctx, patched, client.MergeFrom(measurementDevice)); err != nil {
			// 	return ReconcileResult{Error: err, Stop: true}
			// }
			return ReconcileResult{Stop: true}
		}

		patched := measurementDevice.DeepCopy()
		meta.SetStatusCondition(&patched.Status.Conditions, metav1.Condition{
			Type:               "JobSucceeded",
			Status:             metav1.ConditionTrue,
			Reason:             "JobSucceeded",
			ObservedGeneration: measurementDevice.Generation,
		})
		if err := r.Patch(ctx, patched, client.MergeFrom(measurementDevice)); err != nil {
			return ReconcileResult{Error: err, Stop: true}
		}

		return ReconcileResult{}
	} else {
		err := fmt.Errorf("MeasurementDevice owns multiple owned jobs. This should not be possible.")
		return ReconcileResult{Error: err, Stop: true}
	}
}

func (r *MeasurementDeviceReconciler) reconcileSNMPFile(ctx context.Context, measurementDevice *chantico.MeasurementDevice) ReconcileResult {
	/*
		update the hash of the snmp file in annotations or in status
	*/

	return ReconcileResult{}
}

func (r *MeasurementDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chantico.MeasurementDevice{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

func isJobSuccessful(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

/*
kind: MeasurementDevice
metadata:
  name: voorbeeld
  namespace: chantico
spec:
  ...
status:
  ...


kind: Job
metadata:
	annotations:
	ownerReferences:
	- controller: true
	  kind: MeasurementDevice
	  name: voorbeeld
	  namespace: chantico
spec:
status:


*/
