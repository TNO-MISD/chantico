package measurementdevice

import (
	"fmt"
	"strings"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	img "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/internal/images"
	vol "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/internal/volumes"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	snmpDir       = "snmp"
	snmpYmlDir    = "snmp/yml"
	snmpConfigDir = "snmp/config"
	snmpMibsDir   = "snmp/mibs"
)

func MakeJob(measurementDevice chantico.MeasurementDevice, timestamp int) *batchv1.Job {
	volume, _ := vol.GetChanticoVolume()

	containers := []corev1.Container{
		{
			Name:  "create-snmp-config",
			Image: img.SnmpGenerator,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      vol.ChanticoVolumeMount,
					MountPath: "/opt/snmp.yml",
					SubPath:   fmt.Sprintf("%s/snmp.yml", snmpYmlDir),
				},
				{
					Name:      vol.ChanticoVolumeMount,
					MountPath: "/opt/generator.yml",
					SubPath:   getGeneratorPath(timestamp),
				},
				{
					Name:      vol.ChanticoVolumeMount,
					MountPath: "/opt/mibs",
					SubPath:   snmpMibsDir,
				},
			},
		},
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      measurementDevice.Status.JobName,
			Namespace: measurementDevice.ObjectMeta.Namespace,
		},

		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers:    containers,
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes:       []corev1.Volume{volume},
				},
			},
		},
	}
}

func getGeneratorPath(timestamp int) string {
	return fmt.Sprintf("%s/generator-%d.yml", snmpConfigDir, timestamp)
}

func GenerateSnmpConfig(measurementDevices []chantico.MeasurementDevice) string {
	// TODO: This should support multiple authentication methods
	snmpConfig := `
auths:
  public_v3:
  version: 3
  username: guest
modules:
`

	for _, device := range measurementDevices {
		walkTemplate := "  %s:\n    walk: [%s]\n    auth: public_v3\n"
		snmpConfig += fmt.Sprintf(walkTemplate, device.GetName(), strings.Join(device.Spec.Walks, ","))
	}
	return snmpConfig
}
