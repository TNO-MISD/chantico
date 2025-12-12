package datacenterresource

import (
	"testing"

	chantico "chantico/api/v1alpha1"
)

func TestUpdateState(t *testing.T) {
	testCases := map[string]struct {
		Resource *chantico.DataCenterResource
		Expected string
	}{
		"empty state": {
			Resource: &chantico.DataCenterResource{
				Status: chantico.DataCenterResourceStatus{
					State: "",
				},
			},
			Expected: StateInit,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			UpdateState(tc.Resource)
			if tc.Resource == nil {
				return
			}
			if tc.Resource.Status.State != tc.Expected {
				t.Errorf("UpdateState(%#v) = %#v, want %#v", tc.Resource, tc.Resource.Status.State, tc.Expected)
			}
		})
	}
}
