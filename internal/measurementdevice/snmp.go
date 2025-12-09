package measurementdevice

import (
	"fmt"
	"maps"
	"path/filepath"

	"go.yaml.in/yaml/v2"

	chantico "chantico/api/v1alpha1"
	img "chantico/internal/images"
	vol "chantico/internal/volumes"

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

func MakeJob(measurementDevice chantico.MeasurementDevice) *batchv1.Job {
	volume, _ := vol.GetChanticoVolume()

	containers := []corev1.Container{
		{
			Name:  "create-snmp-config",
			Image: img.SnmpGenerator,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      vol.ChanticoVolumeMount,
					MountPath: "/opt/snmp.yml",
					SubPath:   getConfigPath(measurementDevice),
				},
				{
					Name:      vol.ChanticoVolumeMount,
					MountPath: "/opt/generator.yml",
					SubPath:   getGeneratorPath(measurementDevice),
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

func getGeneratorPath(measurementDevice chantico.MeasurementDevice) string {
	return filepath.Join(
		snmpConfigDir,
		fmt.Sprintf("generator_%s.yml", measurementDevice.Name),
	)
}

func getConfigPath(measurementDevice chantico.MeasurementDevice) string {
	return filepath.Join(
		snmpConfigDir,
		fmt.Sprintf("config_%s.yml", measurementDevice.Name),
	)
}

type generatorModule struct {
	Walk []string `yaml:"walk"`
}

type snmpGeneratorConfig struct {
	Auths   map[string]chantico.Auth   `yaml:"auths"`
	Modules map[string]generatorModule `yaml:"modules"`
}

func GenerateSNMPGeneratorConfig(measurementDevice chantico.MeasurementDevice) (string, error) {
	modules := map[string]generatorModule{}
	modules[measurementDevice.Name] = generatorModule{Walk: measurementDevice.Spec.Walks}

	auths := map[string]chantico.Auth{}
	auths[measurementDevice.Name] = measurementDevice.Spec.Auth
	measurementDeviceSNMPConfig := snmpGeneratorConfig{Auths: auths, Modules: modules}

	out, err := yaml.Marshal(measurementDeviceSNMPConfig)
	return string(out), err
}

type snmpConfig struct {
	Auths   map[string]chantico.Auth `yaml:"auths"`
	Modules map[string]any           `yaml:"modules"`
}

func MergeSNMPConfigs(fileContents [][]byte) (string, error) {
	acc := snmpConfig{Auths: map[string]chantico.Auth{}, Modules: map[string]any{}}
	for _, fileContent := range fileContents {
		snmpconfig := snmpConfig{Auths: map[string]chantico.Auth{}, Modules: map[string]any{}}
		err := yaml.Unmarshal(fileContent, &snmpconfig)
		if err != nil {
			return "", err
		}
		maps.Copy(acc.Auths, snmpconfig.Auths)
		maps.Copy(acc.Modules, snmpconfig.Modules)
	}
	out, err := yaml.Marshal(acc)
	if err != nil {
		return "", err
	}
	return string(out), err
}
