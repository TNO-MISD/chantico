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
	ph "chantico/internal/patch"

	"context"
	"log"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// In this context, "Pure" means "does not modify kubernetes cluster resources"
const (
	ActionFunctionIO = iota
	ActionFunctionPure
)

type ActionResult struct {
	ctrl.Result
	ph.PatchType
}

type ActionFuntion struct {
	Type int
	Pure func(
		*chantico.DataCenterResource,
	) *ActionResult
	IO func(
		context.Context,
		client.Client,
		*chantico.DataCenterResource,
	) *ActionResult
}

var ActionMap = map[string][]ActionFuntion{
	StateInit: {
		ActionFuntion{Type: ActionFunctionPure, Pure: InitializeFinalizer},
	},
	StateEntry: {},

	StatePendingPostgresUpdate:   {},
	StateSucceededPostgresUpdate: {},

	StateDelete: {
		ActionFuntion{Type: ActionFunctionPure, Pure: UpdateFinalizer},
	},

	StateValidationFailed: {},
	StateEnd:              {},
}

func ExecuteActions(
	ctx context.Context,
	kubernetesClient client.Client,
	dataCenterResource *chantico.DataCenterResource,
	patch *ph.PatchHelper,
) *ActionResult {
	var result *ActionResult = nil
	actionFunctions := ActionMap[dataCenterResource.Status.State]
	for i, actionFunction := range actionFunctions {
		log.Printf("Start step %d, status: %s\n", i, dataCenterResource.Status.State)
		switch actionFunction.Type {
		case ActionFunctionPure:
			result = actionFunction.Pure(dataCenterResource)
		case ActionFunctionIO:
			result = actionFunction.IO(ctx, kubernetesClient, dataCenterResource)
		}

		if result != nil {
			patch.Patch(result.PatchType)
		}
		if dataCenterResource.Status.State == StateValidationFailed {
			break
		}
	}
	return result
}

func InitializeFinalizer(
	dataCenterResource *chantico.DataCenterResource,
) *ActionResult {
	if slices.Contains(dataCenterResource.ObjectMeta.Finalizers, chantico.DataCenterResourceGraphFinalizer) {
		return &ActionResult{PatchType: ph.PatchResourceNone}
	}
	dataCenterResource.ObjectMeta.Finalizers = append(dataCenterResource.ObjectMeta.Finalizers, chantico.DataCenterResourceGraphFinalizer)
	log.Printf("Added finalizer: %#v", dataCenterResource.ObjectMeta.Finalizers)
	return &ActionResult{PatchType: ph.PatchResource}
}

func UpdateFinalizer(
	dataCenterResource *chantico.DataCenterResource,
) *ActionResult {
	if dataCenterResource.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil
	}
	accumulator := []string{}
	for _, f := range dataCenterResource.ObjectMeta.Finalizers {
		if f != chantico.DataCenterResourceGraphFinalizer {
			accumulator = append(accumulator, f)
		}
	}
	dataCenterResource.ObjectMeta.Finalizers = accumulator
	return &ActionResult{PatchType: ph.PatchResource}
}
