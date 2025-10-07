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
	)
	IO func(
		context.Context,
		ctrl.Request,
		*chantico.MeasurementDevice,
		[]chantico.MeasurementDevice,
	)
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
) {
	actionFunctions := ActionMap[state]
	for _, actionFunction := range actionFunctions {
		switch actionFunction.Type {
		case ActionFunctionPure:
			{
				actionFunction.Pure(measurementDevice, measurementDevices)
			}
		case ActionFunctionIO:
			{
				actionFunction.IO(ctx, req, measurementDevice, measurementDevices)
			}
		}
	}
}

func InitializeFinalizer(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	if slices.Contains(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer) {
		return
	}
	measurementDevice.ObjectMeta.Finalizers = append(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer)
}

func UpdateFinalizer(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	if measurementDevice.ObjectMeta.DeletionTimestamp.IsZero() {
		return
	}
	accumulator := []string{}
	for _, f := range measurementDevice.ObjectMeta.Finalizers {
		if f != chantico.SNMPUpdateFinalizer {
			accumulator = append(accumulator, f)
		}
	}
	measurementDevice.ObjectMeta.Finalizers = accumulator
}

func UpdateModification(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	measurementDevice.Status.UpdateTime = metav1.Time{Time: time.Now()}.Format(time.RFC3339)
	measurementDevice.Status.UpdateGeneration = measurementDevice.ObjectMeta.Generation
}

func AssessLeader(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	// TODO: Implement the logic of AssessLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
}

func ElectLeader(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	// TODO: Implement the logic of ElectLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
	panic("Not implemented yet")
}

func RequeueWithDelay(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	// TODO: Figure out requeuing strategy, might need a redesign
	panic("Not implemented yet")
}

func UpdateSNMPConfig(
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Job launching
	panic("Not implemented yet")
}

func ReloadSNMPService(
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
) {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Deployment reload
	panic("Not implemented yet")
}
