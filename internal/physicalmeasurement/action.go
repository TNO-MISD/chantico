package physicalmeasurement

import (
	"bytes"
	chantico "chantico/api/v1alpha1"
	sqlhelper "chantico/chantico/sql-helper"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
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
	StateInit: {
		ActionFuntion{Type: ActionFunctionIO, IO: UpdatePrometheus},
	},
	StateCompleted: {},
	StateFailed:    {},
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

	config := PrometheusConfig{}
	config.BuildFromPhysicalMeasurementMap(physicalMeasurementMap)

	yamlBytes, _ := yaml.Marshal(config)
	err := os.WriteFile("/tmp/chantico-volume-mount/prometheus/yml/prometheus.yml", yamlBytes, 0644)
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

	// Use env var
	path := os.Getenv("CHANTICO_PROMETHEUS_PORT_9090_TCP_ADDR") + ":9090"
	err = ReloadPrometheus(path)
	if err != nil {
		physicalMeasurement.Status.State = StateFailed
		physicalMeasurement.Status.ErrorMessage = err.Error()
		_ = c.Status().Update(ctx, physicalMeasurement)
		return &ctrl.Result{}
	}

	return &ctrl.Result{}
}

func ReloadPrometheus(prometheusURL string) error {
	reloadURL := prometheusURL + "/-/reload"
	req, err := http.NewRequest(http.MethodPost, reloadURL, bytes.NewBuffer(nil))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prometheus reload failed: status %d", resp.StatusCode)
	}

	return nil
}
