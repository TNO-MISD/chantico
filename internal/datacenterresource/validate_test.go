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
		Resource                 *chantico.DataCenterResource
		Resources                []chantico.DataCenterResource
		ExpectedVisited          []chantico.DataCenterResource
		ExpectedError            error
		ExpectedInvolvedResource string
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
			Resources:                []chantico.DataCenterResource{},
			ExpectedVisited:          []chantico.DataCenterResource{},
			ExpectedError:            nil,
			ExpectedInvolvedResource: "",
		},
		"creates resource with acyclic dependency": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "baremetal",
					Parent: []string{"bar"},
				},
			},
			Resources: []chantico.DataCenterResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "baremetal",
					Parent: []string{"bar"},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			}},
			ExpectedVisited: []chantico.DataCenterResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bar",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			}},
			ExpectedError:            nil,
			ExpectedInvolvedResource: "",
		},
		"creates resource with convergent dependency": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vm1",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "vm",
					Parent: []string{"bm1", "bm2"},
				},
			},
			Resources: []chantico.DataCenterResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pdu1",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "pdu2",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "bm1",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "baremetal",
					Parent: []string{"pdu1", "pdu2"},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "bm2",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "baremetal",
					Parent: []string{"pdu1", "pdu2"},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "vm1",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "vm",
					Parent: []string{"bm1", "bm2"},
				},
			}},
			ExpectedVisited: []chantico.DataCenterResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "bm1",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "baremetal",
					Parent: []string{"pdu1", "pdu2"},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "bm2",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "baremetal",
					Parent: []string{"pdu1", "pdu2"},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "pdu1",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name: "pdu2",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{},
				},
			}},
			ExpectedError:            nil,
			ExpectedInvolvedResource: "",
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
			Resources:                []chantico.DataCenterResource{},
			ExpectedVisited:          []chantico.DataCenterResource{},
			ExpectedError:            ErrorResourceNotFound{InvolvedResource: "bar"},
			ExpectedInvolvedResource: "bar",
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
			ExpectedVisited:          []chantico.DataCenterResource{},
			ExpectedError:            ErrorCycleDetected{InvolvedResource: "bar"},
			ExpectedInvolvedResource: "bar",
		},
		"gives error if a self-reference is found": {
			Resource: &chantico.DataCenterResource{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{"foo"},
				},
			},
			Resources: []chantico.DataCenterResource{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Spec: chantico.DataCenterResourceSpec{
					Type:   "pdu",
					Parent: []string{"foo"},
				},
			}},
			ExpectedVisited:          []chantico.DataCenterResource{},
			ExpectedError:            ErrorCycleDetected{InvolvedResource: "foo"},
			ExpectedInvolvedResource: "foo",
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
			Resources:                []chantico.DataCenterResource{},
			ExpectedVisited:          []chantico.DataCenterResource{},
			ExpectedError:            ErrorUnknownType{Type: "perpetuummobile"},
			ExpectedInvolvedResource: "",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			visited, err, involvedResource := Validate(tc.Resource, tc.Resources, []chantico.PhysicalMeasurement{})
			if !reflect.DeepEqual(visited, tc.ExpectedVisited) || !errors.Is(err, tc.ExpectedError) || involvedResource != tc.ExpectedInvolvedResource {
				t.Errorf("Validate(%#v, %#v) = %#v, %#v, want %#v, %#v\n)", tc.Resource, FormatResources(tc.Resources), FormatResources(visited), err, FormatResources(tc.ExpectedVisited), tc.ExpectedError)
			}
		})
	}
}
