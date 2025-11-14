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
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type PrometheusConfig struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs"`
}

type ScrapeConfig struct {
	JobName        string              `yaml:"job_name"`
	StaticConfigs  []StaticConfig      `yaml:"static_configs"`
	Params         map[string][]string `yaml:"params"`
	MetricsPath    string              `yaml:"metrics_path"`
	ScrapeInterval string              `yaml:"scrape_interval"`
	ScrapeTimeout  string              `yaml:"scrape_timeout"`
	RelabelConfigs []RelabelConfig     `yaml:"relabel_configs"`
}

type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

type RelabelConfig struct {
	SourceLabels []string `yaml:"source_labels,omitempty"`
	TargetLabel  string   `yaml:"target_label"`
	Replacement  string   `yaml:"replacement,omitempty"`
}

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

	config := PrometheusConfig{}
	for deviceID, ips := range physicalMeasurementMap {
		cfg := ScrapeConfig{
			JobName: deviceID,
			StaticConfigs: []StaticConfig{
				{Targets: ips},
			},
			Params: map[string][]string{
				"module": {deviceID},
				"auth":   {"public_v3"},
			},
			MetricsPath:    "/snmp",
			ScrapeInterval: "10s",
			ScrapeTimeout:  "5s",
			RelabelConfigs: []RelabelConfig{
				{SourceLabels: []string{"__address__"}, TargetLabel: "__param_target"},
				{SourceLabels: []string{"__param_target"}, TargetLabel: "instance"},
				{TargetLabel: "__addzress__", Replacement: "chantico-snmp:9116"},
			},
		}
		config.ScrapeConfigs = append(config.ScrapeConfigs, cfg)
	}

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

	err = ReloadPrometheus("http://chantico-prometheus.chantico.svc.cluster.local:9090")
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
