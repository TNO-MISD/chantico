package unit

import (
	"testing"
	"time"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	action "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/internal/action"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestInitializeFinalizer(t *testing.T) {
	testCases := make([](*chantico.MeasurementDevice), 2)
	wants := make([][]string, 2)

	testCases[0] = &chantico.MeasurementDevice{}
	testCases[0].ObjectMeta.Finalizers = []string{}
	wants[0] = []string{chantico.SNMPUpdateFinalizer}

	testCases[1] = &chantico.MeasurementDevice{}
	testCases[1].ObjectMeta.Finalizers = []string{"test"}
	wants[1] = []string{"test", chantico.SNMPUpdateFinalizer}

	for i := range testCases {
		action.InitializeFinalizer(testCases[i], nil)
		if !equalStringSlices(wants[i], testCases[i].ObjectMeta.Finalizers) {
			t.Errorf("Case %d, TARGET: %#v != OBTAINED: %#v\n", i, wants[i], testCases[i].ObjectMeta.Finalizers)
		}
	}
}

func TestUpdateFinalizer(t *testing.T) {
	testCases := make([](*chantico.MeasurementDevice), 1)
	wants := make([][]string, 1)

	testCases[0] = &chantico.MeasurementDevice{}
	testCases[0].ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	testCases[0].ObjectMeta.Finalizers = []string{"test", chantico.SNMPUpdateFinalizer}

	wants[0] = []string{"test"}

	for i := range testCases {
		action.UpdateFinalizer(testCases[i], nil)
		if !equalStringSlices(wants[i], testCases[i].ObjectMeta.Finalizers) {
			t.Errorf("Case %d, TARGET: %#v != OBTAINED: %#v\n", i, wants[i], testCases[i].ObjectMeta.Finalizers)
		}
	}
}

func TestUpdateModification(t *testing.T) {
	testCases := make([](*chantico.MeasurementDevice), 1)
	wants := make([]int64, 1)

	testCases[0] = &chantico.MeasurementDevice{}
	testCases[0].ObjectMeta.Generation = 5

	wants[0] = 5

	for i := range testCases {
		action.UpdateModification(testCases[i], nil)
		if testCases[i].Status.UpdateGeneration != wants[i] {
			t.Errorf("Case %d, TARGET: %#v != OBTAINED: %#v\n", i, wants[i], testCases[i].Status)
		}
	}
}
