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
	vol "chantico/internal/volumes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	chantico "chantico/api/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	util "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"chantico/internal/config"
	"chantico/internal/snmp"
	"crypto/sha256"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Define a custom type for the Action
type StepAction int

// Declare the possible Action values using iota
const (
	ActionContinue StepAction = iota // 0
	ActionRequeue                    // 1
	ActionStop                       // 2
	ActionError                      // 3
)

type StepResult struct {
	Action       StepAction
	RequeueAfter time.Duration
	Err          error
}

func Continue() StepResult {
	return StepResult{
		Action: ActionContinue,
	}
}
func Stop() StepResult {
	return StepResult{
		Action: ActionStop,
	}
}
func Error(err error) StepResult {
	return StepResult{
		Action: ActionError,
		Err:    err,
	}
}
func Requeue(duration time.Duration) StepResult {
	return StepResult{
		Action:       ActionRequeue,
		RequeueAfter: duration,
	}
}

type StepFunction func(context.Context, *chantico.MeasurementDevice) StepResult

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=measurementdevices/finalizers,verbs=create;update;patch

// SnmpGeneratorReconciler reconciles a SNMP generator
type SnmpGeneratorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.Config
}

func (r *SnmpGeneratorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chantico.MeasurementDevice{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

func (r *SnmpGeneratorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
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
		if err := r.reconcileStatus(measurementDevice); err != nil {
			reterr = errors.Join(reterr, err)
		}
		if err := helper.Patch(ctx, measurementDevice); err != nil {
			reterr = errors.Join(reterr, err)
		}
	}()

	steps := []StepFunction{
		r.reconcileDeletion,
		r.ensureFinalizerIsSet,
		r.reconcileGeneratorFile,
		r.reconcileMibFile,
		r.ensureSNMPFileExists,
		r.reconcileSNMPGeneratorJob,
		r.reconcileSNMPFileContent,
		r.setObservedGeneration,
	}
	for _, step := range steps {
		result := step(ctx, measurementDevice)

		switch result.Action {
		case ActionContinue:
			continue
		case ActionStop:
			return ctrl.Result{}, nil
		case ActionError:
			return ctrl.Result{}, result.Err
		case ActionRequeue:
			return ctrl.Result{RequeueAfter: result.RequeueAfter}, nil
		}
	}

	return ctrl.Result{}, nil
}

/*
Determines the "Ready" condition which is shown to users for a general insight into the status. Currently only depends on "Job" condition, but we can expand this. Or even use conditions of the Cluster API.
*/
func (r *SnmpGeneratorReconciler) reconcileStatus(measurementDevice *chantico.MeasurementDevice) error {
	// should use ObservedGeneration for determining up-to-date or old conditions?
	// we should probably also use a global ObservedGeneration (so then we can see what reconcile has been, and whether it matches the conditions)
	jobCondition := meta.FindStatusCondition(measurementDevice.Status.Conditions, string(chantico.ConditionJob))
	if jobCondition == nil {
		measurementDevice.UpdateStatusCondition(chantico.ConditionJob, metav1.ConditionUnknown, chantico.ReasonPending, "Job condition is pending")
		return nil
	}

	measurementDevice.UpdateStatusJobCondition(jobCondition)
	return nil
}

func (r *SnmpGeneratorReconciler) reconcileDeletion(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	if measurementDevice.ObjectMeta.GetDeletionTimestamp() == nil {
		return Continue()
	}

	if !util.ContainsFinalizer(measurementDevice, chantico.MeasurementDeviceFinalizer) {
		// Nothing to do: finalizer already removed, Kubernetes will complete deletion.
		return Stop()
	}

	// 1. Delete owned Jobs. Garbage collection would eventually remove them too but this explicitly does so directly.
	jobs, err := r.getOwnedJobs(ctx, measurementDevice)
	if err != nil {
		return Error(err)
	}
	for i := range jobs {
		job := &jobs[i]
		if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
			return Error(fmt.Errorf("delete owned job %s: %w", job.Name, err))
		}
	}

	// 2. Delete the generator and SNMP output files.
	filesToRemove := []string{
		filepath.Join(r.Config.MountPath, "snmp/generators", fmt.Sprintf("generator-%s.yaml", measurementDevice.GetUID())),
		filepath.Join(r.Config.MountPath, "snmp/snmp", fmt.Sprintf("snmp-%s.yaml", measurementDevice.GetUID())),
	}
	for _, path := range filesToRemove {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return Error(fmt.Errorf("remove %s: %w", path, err))
		}
	}

	// 3. Release the finalizer so Kubertnetes can complete deletion.
	util.RemoveFinalizer(measurementDevice, chantico.MeasurementDeviceFinalizer)
	return Stop()
}

func (r *SnmpGeneratorReconciler) ensureFinalizerIsSet(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	if util.ContainsFinalizer(measurementDevice, chantico.MeasurementDeviceFinalizer) {
		return Continue()
	}
	util.AddFinalizer(measurementDevice, chantico.MeasurementDeviceFinalizer)
	return Stop()
}

func (r *SnmpGeneratorReconciler) generatorFilePath(md *chantico.MeasurementDevice) string {
	return filepath.Join(r.Config.MountPath, "snmp/generators", fmt.Sprintf("generator-%s.yaml", md.GetUID()))
}

func desiredGeneratorConfig(md *chantico.MeasurementDevice) ([]byte, error) {
	return yaml.Marshal(snmp.GeneratorConfig{
		Auths:   map[string]*snmp.GeneratorAuth{md.Name: &md.Spec.Auth},
		Modules: map[string]*snmp.GeneratorModule{md.Name: {Walk: md.Spec.Walks}},
	})
}

func (r *SnmpGeneratorReconciler) reconcileGeneratorFile(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	// log := logf.FromContext(ctx)
	path := r.generatorFilePath(measurementDevice)

	observed, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Error(fmt.Errorf("read generator file %s: %w", path, err))
	}

	desired, err := desiredGeneratorConfig(measurementDevice)
	if err != nil {
		measurementDevice.UpdateStatusCondition(chantico.ConditionGeneratorFile, metav1.ConditionFalse, chantico.ReasonFailed, fmt.Sprintf("failed to marshal generator config: %v", err))
		return Error(err)
	}

	if bytes.Equal(observed, desired) {
		measurementDevice.UpdateStatusCondition(chantico.ConditionGeneratorFile, metav1.ConditionTrue, chantico.ReasonFileWritten, "Generator file is up to date.")
		return Continue()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return Error(err)
	}

	if err := os.WriteFile(path, desired, 0777); err != nil {
		measurementDevice.UpdateStatusCondition(chantico.ConditionGeneratorFile, metav1.ConditionFalse, chantico.ReasonFailed, fmt.Sprintf("failed to write generator file: %v", err))
		return Error(fmt.Errorf("write generator file %s: %w", path, err))
	}

	measurementDevice.UpdateStatusCondition(chantico.ConditionGeneratorFile, metav1.ConditionTrue, chantico.ReasonFileWritten, "Generator file has been generated successfully.")
	return Continue()
}

func (r *SnmpGeneratorReconciler) reconcileMibFile(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	/*
		I think we should be more explicit for MIB files, or directories. This way we can prevent name space collisions.
	*/
	return Continue()
}

func (r *SnmpGeneratorReconciler) ensureSNMPFileExists(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	/*
		We need to have an SNMP file (even if it is empty, it will be filled later by SNMP Generator).
	*/
	// for now create snmp dir, for some reason this is now done from an init container...
	// Chantico CR, then the Chantico controller will create the folders

	pathToFile := filepath.Join(r.Config.MountPath, "snmp/snmp", fmt.Sprintf("snmp-%s.yaml", measurementDevice.GetUID()))

	_, err := os.ReadFile(pathToFile)
	if err == nil {
		// file exists, awesome
		return Continue()
	}
	if !errors.Is(err, fs.ErrNotExist) {
		// another error, maybe permissions, or smth
		return Error(err)
	}

	// create file
	dir := filepath.Dir(pathToFile)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return Error(err)
	}
	err = os.WriteFile(pathToFile, []byte{}, 0777)
	if err != nil {
		return Error(err)
	}
	return Continue()
}

// Check presence of job and creates one if necessary. Also checks the status of the job, and updates conditions accordingly.
// Preconditions:
// - There is no job for creating the SNMP config yet.
// Result:
// - SNMP config created with job.
func (r *SnmpGeneratorReconciler) reconcileSNMPGeneratorJob(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	ownedJobs, err := r.getOwnedJobs(ctx, measurementDevice)
	if err != nil {
		return Error(err)
	}

	if len(ownedJobs) == 0 {
		job, err := r.buildGeneratorJob(measurementDevice)
		if err != nil {
			return Error(err)
		}
		if err := controllerutil.SetControllerReference(measurementDevice, job, r.Scheme); err != nil {
			return Error(err)
		}

		if err := r.Create(ctx, job); err != nil {
			return Error(err)
		}

		measurementDevice.UpdateStatusCondition(chantico.ConditionJob, metav1.ConditionUnknown, chantico.ReasonPending, "Job condition is pending")
		return Stop()
	} else if len(ownedJobs) == 1 {
		job := ownedJobs[0]

		annotations := job.GetAnnotations()

		observedGeneration, exists := annotations["measurementdevice.generation.chantico"]
		if !exists {
			err := fmt.Errorf("Annotation has not been set for job. Should not be possible.")
			return Error(err)
		}
		desiredGeneration := strconv.FormatInt(measurementDevice.GetGeneration(), 10)
		if observedGeneration != desiredGeneration {
			// job is not up to date
			if err := r.Delete(ctx, &job); err != nil {
				err := fmt.Errorf("Could not delete job.")
				return Error(err)
			}
			return Stop()
		}

		if isJobSuccessful(&job) {
			measurementDevice.UpdateStatusCondition(chantico.ConditionJob, metav1.ConditionTrue, chantico.ReasonJobSucceeded, "SNMP Generator Job has completed successfully.")
			return Continue()
		} else if isJobFailed(&job) {
			measurementDevice.UpdateStatusCondition(chantico.ConditionJob, metav1.ConditionFalse, chantico.ReasonJobFailed, "SNMP Generator Job has failed.")
			return Stop()
		} else {
			measurementDevice.UpdateStatusCondition(chantico.ConditionJob, metav1.ConditionUnknown, chantico.ReasonJobPending, "SNMP Generator Job is pending.")
			return Stop()

		}
	} else {
		err := fmt.Errorf("MeasurementDevice owns multiple owned jobs. This should not be possible.")
		return Error(err)
	}
}

func (r *SnmpGeneratorReconciler) reconcileSNMPFileContent(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	pathToFile := filepath.Join(os.Getenv(vol.ChanticoVolumeLocationEnv), "snmp/snmp", fmt.Sprintf("snmp-%s.yaml", measurementDevice.GetUID()))
	config, err := os.ReadFile(pathToFile)
	if err != nil {
		return Error(err)
	}

	configSha := sha256.Sum256(config)
	configHash := hex.EncodeToString(configSha[:])

	if measurementDevice.Status.ConfigHash == configHash {
		measurementDevice.UpdateStatusCondition(chantico.ConditionConfig, metav1.ConditionTrue, chantico.ReasonSucceeded, "ConfigHash matches with SNMP configuration")
		return Continue()
	}

	measurementDevice.Status.ConfigHash = configHash
	measurementDevice.UpdateStatusCondition(chantico.ConditionConfig, metav1.ConditionTrue, chantico.ReasonSynced, "ConfigHash has been updated to match with SNMP configuration")

	return Stop()
}

func (r *SnmpGeneratorReconciler) setObservedGeneration(ctx context.Context, measurementDevice *chantico.MeasurementDevice) StepResult {
	// Completed the reconcilitation
	measurementDevice.Status.ObservedGeneration = measurementDevice.Generation
	return Continue()
}
