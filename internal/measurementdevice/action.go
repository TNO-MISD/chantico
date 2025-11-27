package measurementdevice

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	chantico "chantico/api/v1alpha1"
	pm "chantico/internal/postmortem"
	vol "chantico/internal/volumes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// In that context Pure means does not modify the kubernetes cluster resources
const (
	ActionFunctionIO = iota
	ActionFunctionPure
)

type ActionFuntion struct {
	Type int
	Pure func(
		*chantico.MeasurementDevice,
	) *ctrl.Result
	IO func(
		context.Context,
		ctrl.Request,
		*chantico.MeasurementDevice,
	) *ctrl.Result
}

var ActionMap = map[string][]ActionFuntion{
	StateInit: {
		ActionFuntion{Type: ActionFunctionPure, Pure: InitializeFinalizer},
	},
	StateEntryPoint: {
		ActionFuntion{Type: ActionFunctionPure, Pure: CreateSNMPGenerator},
		ActionFuntion{Type: ActionFunctionIO, IO: UpdateSNMPConfig},
	},
	StateFailed: {},

	StatePendingSNMPConfigUpdate: {
		ActionFuntion{Type: ActionFunctionPure, Pure: RequeueWithDelay},
	},
	StateSucceededSNMPConfigUpdate: {
		ActionFuntion{Type: ActionFunctionPure, Pure: UpdateModification},
		ActionFuntion{Type: ActionFunctionPure, Pure: AssessLeader},
	},

	StatePendingOnLeader: {},
	StateElectedLeader: {
		ActionFuntion{Type: ActionFunctionIO, IO: ReloadSNMPService},
	},

	StatePendingSNMPServiceUpdate: {
		ActionFuntion{Type: ActionFunctionPure, Pure: RequeueWithDelay},
	},
	StateSucceededSNMPServiceUpdate: {
		ActionFuntion{Type: ActionFunctionPure, Pure: UpdateFinalizer},
		ActionFuntion{Type: ActionFunctionPure, Pure: ElectLeader},
	},
}

func ExecuteActions(
	state string,
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	result := &ctrl.Result{}
	actionFunctions := ActionMap[state]
	for _, actionFunction := range actionFunctions {
		switch actionFunction.Type {
		case ActionFunctionPure:
			{
				result = actionFunction.Pure(measurementDevice)
			}
		case ActionFunctionIO:
			{
				result = actionFunction.IO(ctx, req, measurementDevice)
			}
		}
		if result != nil {
			return result
		}
	}
	return result
}

func InitializeFinalizer(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	if slices.Contains(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer) {
		return nil
	}
	measurementDevice.ObjectMeta.Finalizers = append(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer)
	return nil
}

func UpdateFinalizer(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	if measurementDevice.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}
	accumulator := []string{}
	for _, f := range measurementDevice.ObjectMeta.Finalizers {
		if f != chantico.SNMPUpdateFinalizer {
			accumulator = append(accumulator, f)
		}
	}
	measurementDevice.ObjectMeta.Finalizers = accumulator
	return nil
}

func UpdateModification(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	measurementDevice.Status.UpdateTime = metav1.Time{Time: time.Now()}.Format(time.RFC3339)
	measurementDevice.Status.UpdateGeneration = measurementDevice.ObjectMeta.Generation
	return nil
}

func AssessLeader(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Implement the logic of AssessLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
	return nil
}

func ElectLeader(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Implement the logic of ElectLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
	panic("Not implemented yet")
}

func RequeueWithDelay(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Figure out requeuing strategy, might need a redesign
	return &ctrl.Result{RequeueAfter: chantico.RequeueDelay}
}

func CreateSNMPGenerator(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	generatorYaml, err := GenerateSNMPGeneratorConfig(*measurementDevice)
	if err != nil {
		pm.NewPostMortem(err, measurementDevice)
	}
	generatorPath := fmt.Sprintf(
		"%s/%s/generator-%s.yml",
		os.Getenv(vol.ChanticoVolumeLocationEnv),
		snmpConfigDir,
		string(measurementDevice.GetUID()),
	)
	err = os.WriteFile(generatorPath, []byte(generatorYaml), 0666)
	if err != nil {
		measurementDevice.Status.State = StateFailed
		measurementDevice.Status.ErrorMessage = fmt.Sprintf("Could not write to %s", generatorPath)
	}
	return nil
}

func CreateSNMPDeploymentConfig(
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	// Find files match the config-*.yml format
	configFilesGlobPattern := filepath.Join(
		os.Getenv(vol.ChanticoVolumeLocationEnv),
		snmpConfigDir,
		"config_*.yml",
	)
	configFilePaths, err := filepath.Glob(configFilesGlobPattern)
	if err != nil {
		return nil
	}

	// Create the file contents structure
	fileContents := [][]byte{}
	for _, configFilePath := range configFilePaths {
		fileContent, err := os.ReadFile(configFilePath)
		if err != nil {
			fmt.Printf("Could not load file %s: %s", configFilePath, err)
		}
		fileContents = append(fileContents, fileContent)
	}

	// Merge the data
	mergedSNMPConfig, err := MergeSNMPConfigs(fileContents)
	if err != nil {
		fmt.Printf("Could not create the SNMP deployment config: %s", err)
		return nil
	}
	configSNMPPath := filepath.Join(
		os.Getenv(vol.ChanticoVolumeLocationEnv),
		snmpYmlDir,
		"snmp.yml",
	)
	err = os.WriteFile(
		configSNMPPath,
		[]byte(mergedSNMPConfig),
		0666,
	)
	if err != nil {
		fmt.Printf("Could not write to %s: %s", configSNMPPath, err)
		return nil
	}
	return nil
}

func UpdateSNMPConfig(
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Job launching
	panic("Not implemented yet")
}

func ReloadSNMPService(
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Deployment reload
	panic("Not implemented yet")
}
