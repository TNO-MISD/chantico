package measurementdevice

import (
	"context"
	"slices"
	"time"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ActionFunctionIO = iota
	ActionFunctionPure
)

type ActionFuntion struct {
	Type int
	Pure func(
		*chantico.MeasurementDevice,
		[]chantico.MeasurementDevice,
	) *ctrl.Result
	IO func(
		context.Context,
		ctrl.Request,
		*chantico.MeasurementDevice,
		[]chantico.MeasurementDevice,
	) *ctrl.Result
}

var ActionMap = map[string][]ActionFuntion{
	StateInit: {
		ActionFuntion{Type: ActionFunctionPure, Pure: InitializeFinalizer},
	},
	StateEntryPoint: {
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
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	result := &ctrl.Result{}
	actionFunctions := ActionMap[state]
	for _, actionFunction := range actionFunctions {
		switch actionFunction.Type {
		case ActionFunctionPure:
			{
				result = actionFunction.Pure(measurementDevice, measurementDevices)
			}
		case ActionFunctionIO:
			{
				result = actionFunction.IO(ctx, req, measurementDevice, measurementDevices)
			}
		}
		if result == nil {
			return nil
		}
	}
	return result
}

func InitializeFinalizer(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	if slices.Contains(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer) {
		return nil
	}
	measurementDevice.ObjectMeta.Finalizers = append(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer)
	return nil
}

func UpdateFinalizer(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
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
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	measurementDevice.Status.UpdateTime = metav1.Time{Time: time.Now()}.Format(time.RFC3339)
	measurementDevice.Status.UpdateGeneration = measurementDevice.ObjectMeta.Generation
	return nil
}

func AssessLeader(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Implement the logic of AssessLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
	return nil
}

func ElectLeader(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Implement the logic of ElectLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
	panic("Not implemented yet")
	return nil
}

func RequeueWithDelay(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Figure out requeuing strategy, might need a redesign
	panic("Not implemented yet")
	return nil
}

func UpdateSNMPConfig(
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Job launching
	panic("Not implemented yet")
	return nil
}

func ReloadSNMPService(
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) *ctrl.Result {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Deployment reload
	panic("Not implemented yet")
	return nil
}
