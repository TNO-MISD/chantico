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

package controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PatchHelper struct {
	client client.Client
	ctx    context.Context
	obj    client.Object
	base   client.Object
}

func Initialize(ctx context.Context, c client.Client, obj client.Object) *PatchHelper {
	return &PatchHelper{
		ctx:    ctx,
		client: c,
		obj:    obj,
		base:   obj.DeepCopyObject().(client.Object),
	}
}

func (p *PatchHelper) PatchSpec() error {
	if err := p.client.Patch(p.ctx, p.obj, client.MergeFrom(p.base)); err != nil {
		return err
	}
	p.base = p.obj.DeepCopyObject().(client.Object)
	return nil
}

func (p *PatchHelper) PatchStatus() error {
	if err := p.client.Status().Patch(p.ctx, p.obj, client.MergeFrom(p.base)); err != nil {
		return err
	}
	p.base = p.obj.DeepCopyObject().(client.Object)
	return nil
}
