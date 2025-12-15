/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package datacenterresource

import (
	chantico "chantico/api/v1alpha1"
	"fmt"
)

const (
	StateInit                    = "Init"
	StateEntry                   = "Entry point"
	StateValidationFailed        = "Validation Failed"
	StatePendingPostgresUpdate   = "Pending Postgres Update"
	StateSucceededPostgresUpdate = "Successful Postgres Update"
	StateDelete                  = "Delete"
	StateEnd                     = "End point"
)

func UpdateState(
	datacenterResource *chantico.DataCenterResource,
) {
	// Covers the initialization pathological cases
	if datacenterResource == nil {
		return
	}
	if datacenterResource.Status.UpdateGeneration == 0 {
		datacenterResource.Status.UpdateGeneration = 1
	}

	// Covers lifecycle related changes
	switch {
	case datacenterResource.Status.UpdateGeneration < datacenterResource.ObjectMeta.Generation:
		datacenterResource.Status.State = StateEntry
	case datacenterResource.ObjectMeta.GetDeletionTimestamp() != nil:
		datacenterResource.Status.State = StateDelete
	}

	// Realize the update
	switch datacenterResource.Status.State {
	case "", StateInit:
		datacenterResource.Status.State = StateInit
		datacenterResource.Status.UpdateGeneration = datacenterResource.ObjectMeta.Generation
		return
	case StateEntry:
		datacenterResource.Status.UpdateGeneration = datacenterResource.ObjectMeta.Generation
		return

	case StatePendingPostgresUpdate:
		return
	case StateSucceededPostgresUpdate:
		return
	case StateEnd, StateValidationFailed, StateDelete:
		return
	default:
		SetValidationError(datacenterResource, fmt.Errorf("unknown state"), "")
		return
	}
}

func SetValidationError(
	datacenterResource *chantico.DataCenterResource,
	err error,
	involvedResource string,
) {
	datacenterResource.Status.State = StateValidationFailed
	datacenterResource.Status.ErrorMessage = fmt.Sprintf("validation error: %s", err)
	datacenterResource.Status.ErrorType = fmt.Sprintf("%T", err)
	datacenterResource.Status.InvolvedResource = involvedResource
}

func ClearValidationError(
	datacenterResource *chantico.DataCenterResource,
) {
	datacenterResource.Status.State = StateInit
	datacenterResource.Status.ErrorMessage = ""
	datacenterResource.Status.ErrorType = ""
	datacenterResource.Status.InvolvedResource = ""
}
