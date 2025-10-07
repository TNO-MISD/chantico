package measurementdevice

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"

	chantico "ci.tno.nl/gitlab/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/api/v1alpha1"
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

func GetState(
	measurementDevice *chantico.MeasurementDevice,
	measurementDevices []chantico.MeasurementDevice,
	snmpJob *batchv1.Job,
	snmpExporterDeployment *appsv1.Deployment,
) string {
	if measurementDevice == nil {
		return StateEndPoint
	}

	switch measurementDevice.Status.State {
	case "":
		return StateInit
	default:
		panic("Not implemented yet")
	}
}
