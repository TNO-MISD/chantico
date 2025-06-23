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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chanticov1alpha1 "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
)

// PhysicalMeasurementReconciler reconciles a PhysicalMeasurement object
type PhysicalMeasurementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=physicalmeasurements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=physicalmeasurements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=chantico.ci.tno.nl,resources=physicalmeasurements/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PhysicalMeasurement object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *PhysicalMeasurementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the different MeasurementDevices
	measurementDevices := &chanticov1alpha1.MeasurementDeviceList{}
	err := r.List(ctx, measurementDevices)

	if err != nil {
		return ctrl.Result{}, err
	}

	// Get the different PhysicalMeasurements
	physicalMeasurements := &chanticov1alpha1.PhysicalMeasurementList{}
	err = r.List(ctx, physicalMeasurements)

	if err != nil {
		return ctrl.Result{}, err
	}

	// Associates the PhysicalMeasurements to the MeasurementDevices
	physicalMeasurementMap := make(map[string][]string)

	for _, physicalMeasurement := range physicalMeasurements.Items {
		deviceId := physicalMeasurement.Spec.MeasurementDevice
		physicalMeasurementMap[deviceId] = append(
			physicalMeasurementMap[deviceId],
			physicalMeasurement.Spec.Ip,
		)
	}

	jsonData, err := json.Marshal(physicalMeasurementMap)
	fmt.Printf("\n%s\n", jsonData)

	// Generate the scrape config
	var configLines []string
	configLines = append(configLines, "scrape_configs:")
	for deviceId, ips := range physicalMeasurementMap {
		configLines = append(configLines, fmt.Sprintf("  - job_name: \"%s\"", deviceId))
		configLines = append(configLines, "    static_configs:")
		configLines = append(configLines, "      - targets:")
		for _, ip := range ips {
			configLines = append(configLines, fmt.Sprintf("        - \"%s\"", ip))
		}
	}
	configLines = append(configLines, "    scrape_interval: 10s")
	configLines = append(configLines, "    scrape_timeout: 5s")
	configLines = append(configLines, "    relabel_configs:")
	configLines = append(configLines, "      - source_labels: [__address__]")
	configLines = append(configLines, "    target_label: __param_target")
	configLines = append(configLines, "      - source_labels: [__param_target]")
	configLines = append(configLines, "    target_label: instance")
	configLines = append(configLines, "      - target_label: __address__")
	configLines = append(configLines, "    replacement: chantico_snmp:9116")

	fmt.Printf("\n%s\n", strings.Join(configLines, "\n"))

	// Save ID / Measurement in postgres

	// Reload the deployment

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PhysicalMeasurementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.PhysicalMeasurement{}).
		Complete(r)
}
