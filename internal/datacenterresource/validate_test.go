package datacenterresource

import (
	"errors"
	"testing"

	chantico "chantico/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		Resource  *chantico.DataCenterResource
		Resources []chantico.DataCenterResource
		Expected  error
	}{
		"creates resource if empty": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			},
			Resources: []chantico.DataCenterResource{},
			Expected:  nil,
		},
		"gives error if a resource is not found": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{"bar"},
				},
			},
			Resources: []chantico.DataCenterResource{},
			Expected:  ErrorResourceNotFound{Name: "bar"},
		},
		"gives error if a cycle is found": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{"bar"},
				},
			},
			Resources: []chantico.DataCenterResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{"bar"},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{"foo"},
				},
			}},
			Expected: ErrorCycleDetected{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := Validate(tc.Resource, tc.Resources, []chantico.PhysicalMeasurement{})
			if !errors.Is(err, tc.Expected) {
				t.Errorf("Validate(%#v) = %#v, want %#v\n)", tc, err, tc.Expected)
			}
		})
	}
}
