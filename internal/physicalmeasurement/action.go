package physicalmeasurement

import (
	"bytes"
	chantico "chantico/api/v1alpha1"
	sqlhelper "chantico/chantico/sql-helper"
	"context"
	"fmt"
	"net/http"
	"os"
	"slices"
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
	) *ctrl.Result
	IO func(
		context.Context,
		client.Client,
		*chantico.PhysicalMeasurement,
	) *ctrl.Result
}

var ActionMap = map[string][]ActionFuntion{
	StateInit: {
		ActionFuntion{Type: ActionFunctionPure, Pure: InitializeFinalizer},
		ActionFuntion{Type: ActionFunctionIO, IO: UpdatePrometheus},
	},
	StateRunning: {
		ActionFuntion{Type: ActionFunctionIO, IO: UpdatePrometheus},
	},
	StateDeleted: {
		ActionFuntion{Type: ActionFunctionIO, IO: UpdatePrometheus},
	},
	StateFailed:    {},
	StateCompleted: {},
}

func InitializeFinalizer(physicalMeasurement *chantico.PhysicalMeasurement) *ctrl.Result {
	if slices.Contains(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer) {
		return nil
	}
	physicalMeasurement.ObjectMeta.Finalizers = append(physicalMeasurement.ObjectMeta.Finalizers, chantico.PhysicalMeasurementFinalizer)
	return nil
}

func ExecuteActions(
	state string,
	ctx context.Context,
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,

) *ctrl.Result {
	result := &ctrl.Result{}
	actionFunctions := ActionMap[state]
	for _, actionFunction := range actionFunctions {
		switch actionFunction.Type {
		case ActionFunctionPure:
			{
				result = actionFunction.Pure(physicalMeasurement)
			}
		case ActionFunctionIO:
			{
				result = actionFunction.IO(ctx, c, physicalMeasurement)
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
	c client.Client,
	physicalMeasurement *chantico.PhysicalMeasurement,
) *ctrl.Result {
	physicalMeasurement.Status.State = StateRunning
	physicalMeasurement.Status.Generation = physicalMeasurement.ObjectMeta.Generation
	physicalMeasurement.Status.ErrorMessage = ""

	fmt.Printf("\n\n==PhysicalMeasurement: %s==\n", physicalMeasurement.GetName())
	fmt.Printf("STATE: %s\n", physicalMeasurement.Status.State)
	fmt.Printf("Generation: %s\n", strconv.FormatInt(physicalMeasurement.ObjectMeta.Generation, 10))
	fmt.Printf("===\n\n")

	newConfig := CreatePhysicalMeasurementConfig(
		physicalMeasurement.Spec.MeasurementDevice,
		physicalMeasurement.Spec.ResourceIds,
	)
	cfg := MergeWithPrometheusConfig(os.Getenv("PROMETHEUS_CONFIG"), newConfig)

	yamlBytes, _ := yaml.Marshal(cfg)
	err := os.WriteFile(os.Getenv("PROMETHEUS_CONFIG"), yamlBytes, 0644)
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

func DeletePhysicalMeasurementConfig(physicalMeasurement *chantico.PhysicalMeasurement) *ctrl.Result {
	return nil
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
