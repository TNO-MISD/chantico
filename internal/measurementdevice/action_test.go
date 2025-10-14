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
	testCases := map[string]struct {
		Case     *chantico.MeasurementDevice
		Expected []string
	}{
		"empty finalizer": {
			Case: &chantico.MeasurementDevice{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{},
				}},
			Expected: []string{chantico.SNMPUpdateFinalizer},
		},
		"already initialized": {
			Case: &chantico.MeasurementDevice{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"test"},
				}},
			Expected: []string{"test", chantico.SNMPUpdateFinalizer},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			InitializeFinalizer(tc.Case, nil)
			if !equalStringSlices(tc.Expected, tc.Case.ObjectMeta.Finalizers) {
				t.Errorf("InitializeFinalizer(%#v) = %#v, want %#v\n", tc, tc.Case.ObjectMeta.Finalizers, tc.Expected)
			}
		})
	}
}

func TestUpdateFinalizer(t *testing.T) {
	testCases := map[string]struct {
		Case     *chantico.MeasurementDevice
		Expected []string
	}{
		"removes SNMPUpdateFinalizer on deletion": {
			Case: &chantico.MeasurementDevice{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{"test", chantico.SNMPUpdateFinalizer},
				},
			},
			Expected: []string{"test"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			UpdateFinalizer(tc.Case, nil)
			if !equalStringSlices(tc.Expected, tc.Case.ObjectMeta.Finalizers) {
				t.Errorf("UpdateFinalizer(%#v) = %#v, want %#v\n", tc.Case, tc.Case.ObjectMeta.Finalizers, tc.Expected)
			}
		})
	}
}

func TestUpdateModification(t *testing.T) {
	testCases := map[string]struct {
		Case     *chantico.MeasurementDevice
		Expected int64
	}{
		"copies generation to status": {
			Case: &chantico.MeasurementDevice{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 5,
				},
			},
			Expected: 5,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			UpdateModification(tc.Case, nil)
			if tc.Case.Status.UpdateGeneration != tc.Expected {
				t.Errorf("UpdateModification(%#v) = %#v, want %#v\n", tc.Case, tc.Case.Status.UpdateGeneration, tc.Expected)
			}
		})
	}
}

func TestRequeueWithDelay(t *testing.T) {
	testCases := map[string]struct {
		Device   *chantico.MeasurementDevice
		Devices  []chantico.MeasurementDevice
		Expected time.Duration
	}{
		"default requeue delay": {
			Device:   nil,
			Devices:  nil,
			Expected: chantico.RequeueDelay,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := RequeueWithDelay(tc.Device, tc.Devices)
			if result.RequeueAfter != tc.Expected {
				t.Errorf("RequeueWithDelay() = %#v, want %#v", result.RequeueAfter, tc.Expected)
			}
		})
	}
}

func TestActionMap(t *testing.T) {
	for state, actions := range ActionMap {
		for _, action := range actions {
			t.Run(fmt.Sprintf("action %#v in state %#v", action.Type, state), func(t *testing.T) {
				switch action.Type {
				case ActionFunctionPure:
					if action.IO != nil {
						t.Errorf("Pure action should not have IO: %#v", action)
					}
					if action.Pure == nil {
						t.Errorf("Pure action must have Pure function: %#v", action)
					}
				case ActionFunctionIO:
					if action.IO == nil {
						t.Errorf("IO action must have IO function: %#v", action)
					}
					if action.Pure != nil {
						t.Errorf("IO action should not have Pure function: %#v", action)
					}
				default:
					t.Errorf("Unknown action type: %#v", action.Type)
				}
			})
		}
	}
}
