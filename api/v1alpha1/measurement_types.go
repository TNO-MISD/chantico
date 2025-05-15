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
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"sigs.k8s.io/controller-runtime/pkg/log"

	sqlhelper "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico/sql-helper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MeasurementSpec defines the desired state of Measurement
type MeasurementSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Measurement. Edit measurement_types.go to remove/update
	Name       string `json:"name"`
	IsInternal bool   `json:"isInternal"`
	Protocol   string `json:"protocol"`
	DataSource string `json:"dataSource"`
	Query      string `json:"query"`
}

// MeasurementStatus defines the observed state of Measurement
type MeasurementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Measurement is the Schema for the measurements API
type Measurement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MeasurementSpec   `json:"spec,omitempty"`
	Status MeasurementStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MeasurementList contains a list of Measurement
type MeasurementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Measurement `json:"items"`
}

func (measurement Measurement) Register(ctx context.Context) {
	// Connect to database
	loggerStruct := log.FromContext(ctx)

	db_string := fmt.Sprintf("postgres://ps_user:SecurePassword@%s:%s/ps_db", os.Getenv("POSTGRES_PORT_5432_TCP_ADDR"), os.Getenv("POSTGRES_SERVICE_PORT"))

	loggerStruct.Info(db_string)

	db, err := pgx.Connect(ctx, db_string)
	if err != nil {
		loggerStruct.Info(err.Error())
		return
	}
	defer db.Close(ctx)

	queries := sqlhelper.New(db)

	uuid := new(pgtype.UUID)
	err = uuid.Scan(string(measurement.UID))
	loggerStruct.Info(string(measurement.UID))
	if err != nil {
		loggerStruct.Info(err.Error())
		return
	}
	loggerStruct.Info(uuid.String())
	measurementParams := sqlhelper.CreateMeasurementParams{
		ID:         *uuid,
		Name:       measurement.Spec.Name,
		IsInternal: measurement.Spec.IsInternal,
		Protocol:   measurement.Spec.Protocol,
		DataSource: measurement.Spec.DataSource,
		Query:      measurement.Spec.Query,
	}

	_, err = queries.CreateMeasurement(ctx, measurementParams)
	if err != nil {
		loggerStruct.Info(err.Error())
		return
	}
}

func init() {
	SchemeBuilder.Register(&Measurement{}, &MeasurementList{})
}
