package controller

import (
	chantico "chantico/api/v1alpha1"
	"context"

	batchv1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vol "chantico/internal/volumes"
	"fmt"
	"path/filepath"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func isJobFailed(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isJobSuccessful(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *SnmpGeneratorReconciler) getOwnedJobs(ctx context.Context, measurementDevice *chantico.MeasurementDevice) ([]batchv1.Job, error) {
	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, client.InNamespace(measurementDevice.GetNamespace())); err != nil {
		return nil, err
	}

	// TODO: this can be optimized with indexing (at the manager)
	var ownedJobs []batchv1.Job
	for _, job := range jobList.Items {
		for _, ownerRef := range job.OwnerReferences {
			if ownerRef.UID == measurementDevice.GetUID() {
				ownedJobs = append(ownedJobs, job)
			}
		}
	}
	return ownedJobs, nil
}

func (r *SnmpGeneratorReconciler) buildGeneratorJob(measurementDevice *chantico.MeasurementDevice) (*batchv1.Job, error) {
	volume, err := vol.GetChanticoVolume() // ugly?
	if err != nil {
		return nil, err
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
	backoffLimit := int32(0)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      measurementDevice.GetName(),
			Namespace: measurementDevice.GetNamespace(),
			Annotations: map[string]string{
				"measurementdevice.generation.chantico": strconv.FormatInt(measurementDevice.GetGeneration(), 10),
			},
		},
		Spec: batchv1.JobSpec{

			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "snmp-generator",
							Image: r.Config.Images.SnmpGenerator,
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

	return job, nil
}
