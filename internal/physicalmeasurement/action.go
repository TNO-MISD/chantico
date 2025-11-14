package physicalmeasurement

import (
	chantico "chantico/api/v1alpha1"
	sqlhelper "chantico/chantico/sql-helper"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ActionFunctionIO = iota
	ActionFunctionPure
)

type ActionFuntion struct {
	Type int
	Pure func(
		*chantico.PhysicalMeasurement,
		[]chantico.PhysicalMeasurement,
	) *ctrl.Result
	IO func(
		context.Context,
		ctrl.Request,
		client.Client,
		*chantico.PhysicalMeasurement,
		[]chantico.PhysicalMeasurement,
		[]chantico.MeasurementDevice,
	) *ctrl.Result
}

var ActionMap = map[string][]ActionFuntion{
	StateEmpty: {
		ActionFuntion{Type: ActionFunctionIO, IO: UpdatePrometheus},
	},
	StateRunning: {
		ActionFuntion{Type: ActionFunctionIO, IO: UpdatePrometheus},
	},
	StateCompleted: {
		ActionFuntion{Type: ActionFunctionIO, IO: ReloadDeployment},
	},
	StateFailed: {
		// ActionFuntion{},
	},
	StateReloaded: {
		// ActionFuntion{},
	},
}

func ExecuteActions(
	state string,
	ctx context.Context,
	req ctrl.Request,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,
	physicalMeasurements []chantico.PhysicalMeasurement,
	measurementDevices []chantico.MeasurementDevice,

) *ctrl.Result {
	result := &ctrl.Result{}
	actionFunctions := ActionMap[state]
	for _, actionFunction := range actionFunctions {
		switch actionFunction.Type {
		case ActionFunctionPure:
			{
				result = actionFunction.Pure(physicalMeasurement, physicalMeasurements)
			}
		case ActionFunctionIO:
			{
				result = actionFunction.IO(ctx, req, c, physicalMeasurement, physicalMeasurements, measurementDevices)
			}
		}
		if result != nil {
			return result
		}
	}

	return result
}

func UpdatePrometheus(
	ctx context.Context,
	req ctrl.Request,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,
	physicalMeasurements []chantico.PhysicalMeasurement,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	physicalMeasurement.Status.State = StateRunning
	physicalMeasurement.Status.Generation = physicalMeasurement.ObjectMeta.Generation
	physicalMeasurement.Status.ErrorMessage = ""

	fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
	fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

	// Associates the PhysicalMeasurements to the MeasurementDevices
	physicalMeasurementMap := make(map[string][]string)

	for _, physicalMeasurement := range physicalMeasurements {
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
		configLines = append(configLines, "      - target_label: __addzress__")
		configLines = append(configLines, "        replacement: chantico-snmp:9116")
	}

	err := os.WriteFile("/tmp/chantico-volume-mount/prometheus/yml/prometheus.yml", []byte(strings.Join(configLines, "\n")), 0644)
	if err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = c.Status().Update(ctx, physicalMeasurement)
		return &ctrl.Result{}
	}

	// Save ID / Measurement in postgres
	dbUrl := os.Getenv("PG_DBSTRING")
	db, err := pgx.Connect(ctx, dbUrl)
	if err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = c.Status().Update(ctx, physicalMeasurement)
		return &ctrl.Result{}
	}
	defer db.Close(ctx)

	queries := sqlhelper.New(db)
	var uuid pgtype.UUID
	err = uuid.Scan(string(physicalMeasurement.UID))
	if err != nil {
		fmt.Printf("UID: %s\n", string(physicalMeasurement.UID))
		return &ctrl.Result{}
	}
	physicalMeasurementParams := sqlhelper.UpdatePhysicalMeasurementParams{
		ID:        uuid,
		ServiceID: physicalMeasurement.Spec.ServiceId,
	}
	_, err = queries.UpdatePhysicalMeasurement(ctx, physicalMeasurementParams)
	if err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = c.Status().Update(ctx, physicalMeasurement)
		return &ctrl.Result{}
	}
	physicalMeasurement.Status.State = StateCompleted
	_ = c.Status().Update(ctx, physicalMeasurement)

	return ReloadDeployment(
		ctx,
		req,
		c,
		physicalMeasurement,
		physicalMeasurements,
		measurementDevices,
	)
}

func ReloadDeployment(
	ctx context.Context,
	req ctrl.Request,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,
	physicalMeasurements []chantico.PhysicalMeasurement,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	deployment := &appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Name: "chantico-prometheus", Namespace: "chantico"}, deployment)
	if err != nil {
		fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
		fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
		fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
		fmt.Printf("===")
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = c.Status().Update(ctx, physicalMeasurement)
		return &ctrl.Result{}
	}

	deployment.Spec.Template.Annotations["reloadedAt"] = time.Now().Format(time.RFC3339)
	if deployment.Status.CollisionCount != nil && *deployment.Status.CollisionCount > 0 {
		return &ctrl.Result{RequeueAfter: 10 * time.Second}
	}
	if deployment.Status.ReadyReplicas < *(deployment.Spec.Replicas) {
		return &ctrl.Result{RequeueAfter: 10 * time.Second}
	}

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}

	err = c.Update(ctx, deployment)
	if err != nil {
		fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
		fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
		fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
		fmt.Printf("===")
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = c.Status().Update(ctx, physicalMeasurement)
		_ = c.Status().Update(ctx, physicalMeasurement)
		return &ctrl.Result{}
	}

	physicalMeasurement.Status.State = StateReloaded
	_ = c.Status().Update(ctx, physicalMeasurement)

	return &ctrl.Result{}
}
