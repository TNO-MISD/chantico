package measurementdevice

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
)

func TestGetState(t *testing.T) {
	nbTest := 1
	measurementDevices := make([]chantico.MeasurementDevice, nbTest)
	measurementDevicesList := make([][]chantico.MeasurementDevice, nbTest)
	jobs := make([]batchv1.Job, nbTest)
	deployments := make([]appsv1.Deployment, nbTest)
	wants := make([]string, nbTest)

	measurementDevices[0] = chantico.MeasurementDevice{}
	measurementDevices[0].Status.State = ""
	wants[0] = StateInit

	for i := range nbTest {
		measurementDevicesIteration := measurementDevices[i]
		measurementDevicesListIteration := measurementDevicesList[i]
		jobsIteration := jobs[i]
		deploymentsIteration := deployments[i]
		wantsIteration := wants[i]
		resultIteration := GetState(
			&measurementDevicesIteration,
			measurementDevicesListIteration,
			&jobsIteration,
			&deploymentsIteration,
		)
		if wantsIteration != resultIteration {
			t.Errorf("Case %d, TARGET: %#v != OBTAINED: %#v\n", i, wantsIteration, resultIteration)
		}
	}
}
