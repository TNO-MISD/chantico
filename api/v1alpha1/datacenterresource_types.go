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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DataCenterResourceSpec defines the desired state of DataCenterResource
type DataCenterResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	Type string `json:"type"`

	// +optional
	PhysicalMeasurements []string `json:"physicalMeasurements,omitempty"`
	// +optional
	Parent string `json:"parent,omitempty"`
}

// DataCenterResourceStatus defines the observed state of DataCenterResource.
type DataCenterResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	State            string `json:"state,omitempty"`
	UpdateTime       string `json:"updateTime,omitempty"`
	UpdateGeneration int64  `json:"updateGeneration,omitempty"`
	ErrorMessage     string `json:"errorMessage,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DataCenterResource is the Schema for the datacenterresources API
type DataCenterResource struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of DataCenterResource
	// +required
	Spec DataCenterResourceSpec `json:"spec"`

	// status defines the observed state of DataCenterResource
	// +optional
	Status DataCenterResourceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// DataCenterResourceList contains a list of DataCenterResource
type DataCenterResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DataCenterResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataCenterResource{}, &DataCenterResourceList{})
}
