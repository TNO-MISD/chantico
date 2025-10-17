package measurementdevice

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
)

func TestGetState(t *testing.T) {
	testCases := map[string]struct {
		MeasurementDevice     *chantico.MeasurementDevice
		MeasurementDeviceList []chantico.MeasurementDevice
		Job                   batchv1.Job
		Deployment            appsv1.Deployment
		Expected              string
	}{
		"empty state": {
			MeasurementDevice: &chantico.MeasurementDevice{
				Status: chantico.MeasurementDeviceStatus{
					State: "",
				},
			},
			Expected: StateInit,
		},
		"nil device": {
			MeasurementDevice: nil,
			Expected:          StateEndPoint,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := GetState(
				tc.MeasurementDevice,
				tc.MeasurementDeviceList,
				&tc.Job,
				&tc.Deployment,
			)
			if result != tc.Expected {
				t.Errorf("GetState(%#v) = %#v, want %#v", tc.MeasurementDevice, result, tc.Expected)
			}
		})
	}
}
