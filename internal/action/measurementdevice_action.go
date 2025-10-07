package action

import (
	"context"
	"slices"
	"time"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
	controller "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/internal/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ActionInitializeFinalizer = iota
	ActionUpdateFinalizer
	ActionUpdateModification
	ActionElectLeader
	ActionAssessLeader
	ActionRequeueWithDelay
	SideEffectUpdateSNMPConfig
	SideEffectReloadSNMPService
)

const (
	StateInit                       = "Init"
	StateEntryPoint                 = "EntryPoint"
	StateElectedLeader              = "ElectedLeader"
	StatePendingOnLeader            = "PendingOnLeader"
	StatePendingSNMPConfigUpdate    = "PendingSNMPConfigUpdate"
	StateSucceededSNMPConfigUpdate  = "SucceededSNMPConfigUpdate"
	StatePendingSNMPServiceUpdate   = "PendingSNMPServiceUpdate"
	StateSucceededSNMPServiceUpdate = "StateSucceededSNMPServiceUpdate"
	StateFailed                     = "Failed"
	StateEndPoint                   = "EndPoint"
)

var ActionMap = map[string][]int{
	StateInit:       {ActionInitializeFinalizer},
	StateEntryPoint: {SideEffectUpdateSNMPConfig},
	StateFailed:     {},

	StatePendingSNMPConfigUpdate:   {ActionRequeueWithDelay},
	StateSucceededSNMPConfigUpdate: {ActionUpdateModification, ActionAssessLeader},

	StatePendingOnLeader: {},
	StateElectedLeader:   {SideEffectReloadSNMPService},

	StatePendingSNMPServiceUpdate:   {ActionRequeueWithDelay},
	StateSucceededSNMPServiceUpdate: {ActionUpdateFinalizer, ActionElectLeader},
}

func InitializeFinalizer(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
) {
	if slices.Contains(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer) {
		return
	}
	measurementDevice.ObjectMeta.Finalizers = append(measurementDevice.ObjectMeta.Finalizers, chantico.SNMPUpdateFinalizer)
}

func UpdateFinalizer(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
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
	measurementDevices [](*chantico.MeasurementDevice),
) {
	measurementDevice.Status.UpdateTime = metav1.Time{Time: time.Now()}.Format(time.RFC3339)
	measurementDevice.Status.UpdateGeneration = measurementDevice.ObjectMeta.Generation
}

func AssessLeader(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
) {
	// TODO: Implement the logic of AssessLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
}

func ElectLeader(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
) {
	// TODO: Implement the logic of ElectLeader based on and UpdateTime, UpdateGeneration
	// TODO: Write test associated
	panic("Not implemented yet")
}

func RequeueWithDelay(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
) {
	// TODO: Figure out requeuing strategy, might need a redesign
	panic("Not implemented yet")
}

func UpdateSNMPConfigSideEffect(
	r *controller.MeasurementDeviceReconciler,
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
) {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Job launching
	panic("Not implemented yet")
}

func ReloadSNMPServiceSideEffect(
	r *controller.MeasurementDeviceReconciler,
	ctx context.Context,
	req ctrl.Request,
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices [](*chantico.MeasurementDevice),
) {
	// TODO: Separate cleanly the generalizable part of the Kubernetes Deployment reload
	panic("Not implemented yet")
}
