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

// MeasurementDeviceSpec defines the desired state of MeasurementDevice
type MeasurementDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MeasurementDevice. Edit measurementdevice_types.go to remove/update
	Walks []string `json:"walks"`
}

// MeasurementDeviceStatus defines the observed state of MeasurementDevice
type MeasurementDeviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MeasurementDevice is the Schema for the measurementdevices API
type MeasurementDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MeasurementDeviceSpec   `json:"spec,omitempty"`
	Status MeasurementDeviceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MeasurementDeviceList contains a list of MeasurementDevice
type MeasurementDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MeasurementDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MeasurementDevice{}, &MeasurementDeviceList{})
}
