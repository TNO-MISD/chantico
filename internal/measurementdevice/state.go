package measurementdevice

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"

	chantico "chantico/api/v1alpha1"
)

const (
	StateInit                      = "Init"
	StateEntryPoint                = "Entry Point"
	StatePendingSNMPConfigUpdate   = "Pending SNMP Config Update"
	StateSucceededSNMPConfigUpdate = "SucceededSNMPConfigUpdate"
	StateFailed                    = "Failed"
	StateEndPoint                  = "End Point"
	StateDelete                    = "StateDelete"

	StatePendingSNMPServiceUpdate   = "PendingSNMPServiceUpdate"
	StateSucceededSNMPServiceUpdate = "StateSucceededSNMPServiceUpdate"
)

func GetState(
	measurementDevice *chantico.MeasurementDevice,
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
