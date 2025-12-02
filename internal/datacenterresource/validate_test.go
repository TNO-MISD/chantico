package datacenterresource

import (
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
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := Validate(tc.Resource, tc.Resources, []chantico.PhysicalMeasurement{})
			if err != tc.Expected {
				t.Errorf("Validate(%#v) = %#v, want %#v\n)", tc, err, tc.Expected)
			}
		})
	}
}
