package measurementdevice

import (
	"context"
	"testing"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"gopkg.in/yaml.v3"
)

func TestMakeJob(t *testing.T) {
	// This is an experiment with a kubernetes fake client for test purposes
	// TODO: make relevant tests
	client := fake.NewSimpleClientset()

	// Define test metadata
	device := chantico.MeasurementDevice{
		Status:     chantico.MeasurementDeviceStatus{JobName: "foo"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "bar"},
	}

	// Create job object
	job := MakeJob(device, 5)

	// Create job in fake client
	_, err := client.BatchV1().Jobs(device.ObjectMeta.Namespace).Create(
		context.TODO(),
		job,
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
}

func TestGenerateSnmpConfig(t *testing.T) {
	testCases := map[string]struct {
		Case []chantico.MeasurementDevice
	}{
		"empty case": {
			Case: []chantico.MeasurementDevice{},
		},
		"single measurement device": {
			Case: []chantico.MeasurementDevice{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test"},
					Spec:       chantico.MeasurementDeviceSpec{Walks: []string{"foo", "bar"}},
				},
			},
		},
	}

	// Check that the generated yaml is valid
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var out any
			generatedYaml := []byte(GenerateSnmpConfig(tc.Case))
			err := yaml.Unmarshal(generatedYaml, &out)
			if err != nil {
				t.Fatalf("The generated config is not a valid YAML file: \n%s\n", generatedYaml)
			}
		})
	}

}
