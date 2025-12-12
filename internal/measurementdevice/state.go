package measurementdevice

import (
	"log"
	"time"

	chantico "chantico/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
)

const (
	StateInit                      = "Init"
	StateEntryPoint                = "Entry Point"
	StatePendingSNMPConfigUpdate   = "Pending SNMP Config Update"
	StateSucceededSNMPConfigUpdate = "Succeeded SNMP Config Update"
	StatePendingSNMPReload         = "Pending SNMP Config Reload"
	StateFailed                    = "Failed"

	StateEndPoint = "End Point"
	StateDelete   = "StateDelete"
)

func UpdateState(
	measurementDevice *chantico.MeasurementDevice,
	snmpJob *batchv1.Job,
) {
	// Covers the initialization pathological cases
	if measurementDevice == nil {
		return
	}
	if measurementDevice.Status.UpdateGeneration == 0 {
		measurementDevice.Status.UpdateGeneration = 1
	}

	// Covers lifecycle related changes
	switch {
	case measurementDevice.Status.UpdateGeneration < measurementDevice.ObjectMeta.Generation:
		measurementDevice.Status.State = StateEntryPoint
		break
	case measurementDevice.ObjectMeta.GetDeletionTimestamp() != nil:
		measurementDevice.Status.State = StateDelete
		break
	}

	// Realize the update
	switch measurementDevice.Status.State {
	case "", StateInit:
		measurementDevice.Status.State = StateInit
		measurementDevice.Status.UpdateGeneration = measurementDevice.ObjectMeta.Generation
		return
	case StateEntryPoint:
		measurementDevice.Status.UpdateGeneration = measurementDevice.ObjectMeta.Generation
		return

	case StatePendingSNMPConfigUpdate:
		if snmpJob.Status.Succeeded == 1 {
			measurementDevice.Status.State = StateSucceededSNMPConfigUpdate
		} else if snmpJob.Status.Failed > 0 {
			measurementDevice.Status.State = StateFailed
			log.Fatalf("JOB: %#v", snmpJob)
		} else {
			startTime := snmpJob.Status.StartTime
			if startTime == nil {
				break
			}
			now := time.Now()
			if startTime.Time.Add(chantico.SNMPJobTimeout).Before(now) {
				measurementDevice.Status.State = StateFailed
				log.Fatalf("JOB: %v, %v", startTime.Time, now)
			}
		}
		return
	case StateSucceededSNMPConfigUpdate, StatePendingSNMPReload:
		return
	case StateEndPoint, StateFailed, StateDelete:
		return
	default:
		measurementDevice.Status.State = StateFailed
		return
	}
}
