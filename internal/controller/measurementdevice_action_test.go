package controller

import (
	"testing"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
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
	testCases := make([](*chantico.MeasurementDevice), 1)
	testCases[0].ObjectMeta.Finalizers = []string{}

	wants := make([][]string, 1)
	wants[0] = []string{}
	for i := range testCases {
		InitializeFinalizer(testCases[i], nil)
		if equalStringSlices(testCases[i].ObjectMeta.Finalizers, wants[i]) {
			t.Errorf("TARGET: %#v != OBTAINED: %#v", wants[i], testCases[i])
		}
	}
}
