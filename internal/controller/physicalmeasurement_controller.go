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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	chanticov1alpha1 "chantico/api/v1alpha1"
	sqlhelper "chantico/chantico/sql-helper"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PhysicalMeasurementReconciler reconciles a PhysicalMeasurement object
type PhysicalMeasurementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	PhysicalMeasurementStateRunning   = "Running"
	PhysicalMeasurementStateCompleted = "Completed"
	PhysicalMeasurementStateFailed    = "Failed"
	PhysicalMeasurementStateReloaded  = "Reloaded"
)

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
	return ctrl.Result{}, nil

	physicalMeasurement := &chanticov1alpha1.PhysicalMeasurement{}
	err := r.Get(ctx, req.NamespacedName, physicalMeasurement)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
	fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

	if physicalMeasurement.Status.Generation < physicalMeasurement.ObjectMeta.Generation {
		physicalMeasurement.Status.State = ""
	}

	switch physicalMeasurement.Status.State {
	case "":
		return r.UpdatePrometheus(ctx, physicalMeasurement, req)
	case PhysicalMeasurementStateRunning:
		return r.UpdatePrometheus(ctx, physicalMeasurement, req)
	case PhysicalMeasurementStateCompleted:
		return r.reloadDeployment(ctx, physicalMeasurement, req)
	case PhysicalMeasurementStateFailed:
		return ctrl.Result{}, nil
	case PhysicalMeasurementStateReloaded:
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, fmt.Errorf("unknown state: %s", physicalMeasurement.Status.State)
	}
}

func (r *PhysicalMeasurementReconciler) UpdatePrometheus(ctx context.Context, physicalMeasurement *chanticov1alpha1.PhysicalMeasurement, req ctrl.Request) (ctrl.Result, error) {
	// Set the status
	physicalMeasurement.Status.State = PhysicalMeasurementStateRunning
	physicalMeasurement.Status.Generation = physicalMeasurement.ObjectMeta.Generation
	physicalMeasurement.Status.ErrorMessage = ""
	_ = r.Status().Update(ctx, physicalMeasurement)

	fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
	fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

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
		configLines = append(configLines, "    params:")
		configLines = append(configLines, fmt.Sprintf("      module: [%s]", deviceId))
		configLines = append(configLines, "      auth: [public_v3]")
		configLines = append(configLines, "    metrics_path: \"/snmp\"")
		configLines = append(configLines, "    scrape_interval: 10s")
		configLines = append(configLines, "    scrape_timeout: 5s")
		configLines = append(configLines, "    relabel_configs:")
		configLines = append(configLines, "      - source_labels: [__address__]")
		configLines = append(configLines, "        target_label: __param_target")
		configLines = append(configLines, "      - source_labels: [__param_target]")
		configLines = append(configLines, "        target_label: instance")
		configLines = append(configLines, "      - target_label: __address__")
		configLines = append(configLines, "        replacement: chantico-snmp:9116")
	}

	err = os.WriteFile("/tmp/chantico-volume-mount/prometheus/yml/prometheus.yml", []byte(strings.Join(configLines, "\n")), 0644)
	if err != nil {
		physicalMeasurement.Status.State = PhysicalMeasurementStateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, physicalMeasurement)
		return ctrl.Result{}, err
	}

	// Save ID / Measurement in postgres
	dbUrl := os.Getenv("PG_DBSTRING")
	db, err := pgx.Connect(ctx, dbUrl)
	if err != nil {
		physicalMeasurement.Status.State = PhysicalMeasurementStateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, physicalMeasurement)
		return ctrl.Result{}, err
	}
	defer db.Close(ctx)

	queries := sqlhelper.New(db)
	var uuid pgtype.UUID
	err = uuid.Scan(string(physicalMeasurement.UID))
	if err != nil {
		fmt.Printf("UID: %s\n", string(physicalMeasurement.UID))
		return ctrl.Result{}, err
	}
	physicalMeasurementParams := sqlhelper.UpdatePhysicalMeasurementParams{
		ID:        uuid,
		ServiceID: physicalMeasurement.Spec.ServiceId,
	}
	_, err = queries.UpdatePhysicalMeasurement(ctx, physicalMeasurementParams)
	if err != nil {
		physicalMeasurement.Status.State = PhysicalMeasurementStateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, physicalMeasurement)
		return ctrl.Result{}, err
	}
	physicalMeasurement.Status.State = PhysicalMeasurementStateCompleted
	_ = r.Status().Update(ctx, physicalMeasurement)

	return r.reloadDeployment(ctx, physicalMeasurement, req)
}

func (r *PhysicalMeasurementReconciler) reloadDeployment(ctx context.Context, physicalMeasurement *chanticov1alpha1.PhysicalMeasurement, _ ctrl.Request) (ctrl.Result, error) {

	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, client.ObjectKey{Name: "chantico-prometheus", Namespace: "chantico"}, deployment)
	if err != nil {
		fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
		fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
		fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
		fmt.Printf("===")
		physicalMeasurement.Status.State = PhysicalMeasurementStateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, physicalMeasurement)
		return ctrl.Result{}, err
	}

	deployment.Spec.Template.Annotations["reloadedAt"] = time.Now().Format(time.RFC3339)
	if deployment.Status.CollisionCount != nil && *deployment.Status.CollisionCount > 0 {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if deployment.Status.ReadyReplicas < *(deployment.Spec.Replicas) {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	err = r.Update(ctx, deployment)
	if err != nil {
		fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
		fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
		fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
		fmt.Printf("===")
		physicalMeasurement.Status.State = PhysicalMeasurementStateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = r.Status().Update(ctx, physicalMeasurement)
		_ = r.Status().Update(ctx, physicalMeasurement)
		return ctrl.Result{}, err
	}

	physicalMeasurement.Status.State = PhysicalMeasurementStateReloaded
	_ = r.Status().Update(ctx, physicalMeasurement)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PhysicalMeasurementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&chanticov1alpha1.PhysicalMeasurement{}).
		Complete(r)
}
