/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package datacenterresource

import (
	chantico "chantico/api/v1alpha1"
	"testing"
	"time"

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
		Case     *chantico.DataCenterResource
		Expected []string
	}{
		"empty finalizer": {
			Case: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{},
				}},
			Expected: []string{chantico.DataCenterResourceGraphFinalizer},
		},
		"already initialized": {
			Case: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers: []string{"test"},
				}},
			Expected: []string{"test", chantico.DataCenterResourceGraphFinalizer},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			InitializeFinalizer(tc.Case)
			if !equalStringSlices(tc.Expected, tc.Case.ObjectMeta.Finalizers) {
				t.Errorf("InitializeFinalizer(%#v) = %#v, want %#v\n", tc, tc.Case.ObjectMeta.Finalizers, tc.Expected)
			}
		})
	}
}

func TestUpdateFinalizer(t *testing.T) {
	testCases := map[string]struct {
		Case     *chantico.DataCenterResource
		Expected []string
	}{
		"removes DataCenterResourceGraphFinalizer on deletion": {
			Case: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{"test", chantico.DataCenterResourceGraphFinalizer},
				},
			},
			Expected: []string{"test"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			UpdateFinalizer(tc.Case)
			if !equalStringSlices(tc.Expected, tc.Case.ObjectMeta.Finalizers) {
				t.Errorf("UpdateFinalizer(%#v) = %#v, want %#v\n", tc.Case, tc.Case.ObjectMeta.Finalizers, tc.Expected)
			}
		})
	}
}
