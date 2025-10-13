package measurementdevice

import (
	"fmt"
	"testing"
	"time"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
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

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%#v", wants[i]), func(t *testing.T) {
			InitializeFinalizer(tc, nil)
			if !equalStringSlices(wants[i], tc.ObjectMeta.Finalizers) {
				t.Errorf("InitializeFinalizer(%#v) = %#v, want %#v\n", tc, tc.ObjectMeta.Finalizers, wants[i])
			}
		})
	}
}

func TestUpdateFinalizer(t *testing.T) {
	testCases := make([](*chantico.MeasurementDevice), 1)
	wants := make([][]string, 1)

	testCases[0] = &chantico.MeasurementDevice{}
	testCases[0].ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	testCases[0].ObjectMeta.Finalizers = []string{"test", chantico.SNMPUpdateFinalizer}

	wants[0] = []string{"test"}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%#v", wants[i]), func(t *testing.T) {
			UpdateFinalizer(tc, nil)
			if !equalStringSlices(wants[i], tc.ObjectMeta.Finalizers) {
				t.Errorf("UpdateFinalizer(%#v) = %#v, want %#v\n", tc, tc.ObjectMeta.Finalizers, wants[i])
			}
		})
	}
}

func TestUpdateModification(t *testing.T) {
	testCases := make([](*chantico.MeasurementDevice), 1)
	wants := make([]int64, 1)

	testCases[0] = &chantico.MeasurementDevice{}
	testCases[0].ObjectMeta.Generation = 5

	wants[0] = 5

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%#v", wants[i]), func(t *testing.T) {
			UpdateModification(tc, nil)
			if tc.Status.UpdateGeneration != wants[i] {
				t.Errorf("UpdateModification(%#v) = %#v, want %#v\n", tc, tc.Status, wants[i])
			}
		})
	}
}

func TestActionMap(t *testing.T) {
	for state, actions := range ActionMap {
		for _, action := range actions {
			t.Run(fmt.Sprintf("%#v in %#v", action, state), func(t *testing.T) {
				switch action.Type {
				case ActionFunctionPure:
					if action.IO != nil || action.Pure == nil {
						t.Errorf("%#v is not pure", action)
					}
				case ActionFunctionIO:
					if action.IO == nil || action.Pure != nil {
						t.Errorf("%#v is pure", action)
					}
				}
			})
		}
	}
}
