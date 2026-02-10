// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tree

import (
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Builder = &ServiceBuilderImpl{}

type ServiceBuilderImpl struct {
	*ControllerStore
}

func NewServiceBuilder(controllerStore *ControllerStore) Builder {
	return &ServiceBuilderImpl{
		ControllerStore: controllerStore,
	}
}

// --------------------
// GateTree Updates
// --------------------

func (b *ServiceBuilderImpl) ComputeTreeUpdates() {
	b.computeGateTreeUpdates()
}

func (b *ServiceBuilderImpl) computeGateTreeUpdates() {
	for serviceKey, serviceUpdate := range b.ClusterStore.Updates.Services {
		b.computeTreeServiceUpdate(serviceKey, serviceUpdate)
	}
}

func (b *ServiceBuilderImpl) computeTreeServiceUpdate(serviceKey client.ObjectKey, serviceUpdate store.Update[*v1.Service]) {
	treeService := b.GateTree.Services[serviceKey]

	switch serviceUpdate.Status {
	case store.StatusUpserted:
		if treeService != nil {
			treeService.SetAsUpserted(b.Logger, serviceUpdate.NewObject)
		} else {
			treeService = NewService(serviceUpdate.NewObject)
		}
		treeService.SetAsManaged(b.Logger, *b.ControllerStore)
	case store.StatusDeleted:
		if treeService != nil {
			treeService.SetAsDeleted(b.Logger)
		}

		// else nothing to do
		// It did not exists, it's deleted, noop
	}
}

// -----------------------------------------------

func (b *ServiceBuilderImpl) CleanTreeUpdates() {
	cleanTreeUpdates(b.GateTree.Services)
	cleanTreeUpdates(b.UnmanagedGateTree.Services)
}
