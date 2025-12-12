package datacenterresource

import (
	"errors"
	"reflect"
	"testing"

	chantico "chantico/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidate(t *testing.T) {
	testCases := map[string]struct {
		Resource        *chantico.DataCenterResource
		Resources       []chantico.DataCenterResource
		ExpectedVisited []string
		ExpectedError   error
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
			Resources:       []chantico.DataCenterResource{},
			ExpectedVisited: []string{},
			ExpectedError:   nil,
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
			Resources:       []chantico.DataCenterResource{},
			ExpectedVisited: []string{},
			ExpectedError:   ErrorResourceNotFound{Name: "bar"},
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
			ExpectedVisited: []string{"bar"},
			ExpectedError:   ErrorCycleDetected{},
		},
		"gives error if unknown type is found": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "perpetuummobile",
					Parent: []string{},
				},
			},
			Resources:       []chantico.DataCenterResource{},
			ExpectedVisited: []string{},
			ExpectedError:   ErrorUnknownType{Type: "perpetuummobile"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			visited, err := Validate(tc.Resource, tc.Resources, []chantico.PhysicalMeasurement{})
			if !reflect.DeepEqual(visited, tc.ExpectedVisited) || !errors.Is(err, tc.ExpectedError) {
				t.Errorf("Validate(%#v) = %#v, %#v, want %#v, %#v\n)", tc, visited, err, tc.ExpectedVisited, tc.ExpectedError)
			}
		})
	}
}
