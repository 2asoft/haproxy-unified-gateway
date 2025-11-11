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

var _ Builder = &SecretBuilderImpl{}

type SecretBuilderImpl struct {
	ControllerStore
}

func NewSecretBuilder(controllerStore ControllerStore) Builder {
	return &SecretBuilderImpl{
		ControllerStore: controllerStore,
	}
}

// --------------------
// GateTree Updates
// --------------------

func (b *SecretBuilderImpl) ComputeTreeUpdates() {
	b.computeGateTreeUpdates()
}

func (b *SecretBuilderImpl) computeGateTreeUpdates() {
	for secretKey, secretUpdate := range b.ClusterStore.Updates.Secrets {
		b.computeTreeSecretUpdate(secretKey, secretUpdate)
	}
}

func (b *SecretBuilderImpl) computeTreeSecretUpdate(secretKey client.ObjectKey, secretUpdate store.Update[*v1.Secret]) {
	treeSecret := b.GateTree.Secrets[secretKey]

	switch secretUpdate.Status {
	case store.StatusUpserted:
		if treeSecret != nil {
			treeSecret.SetAsUpserted(b.Logger, secretUpdate.NewObject)
		} else {
			treeSecret = NewSecret(secretUpdate.NewObject)
		}
		treeSecret.SetAsManaged(b.Logger, b.ControllerStore)
	case store.StatusDeleted:
		if treeSecret != nil {
			treeSecret.SetAsDeleted(b.Logger)
		}

		// else nothing to do
		// It did not exists, it's deleted, noop
	}
}

// -----------------------------------------------

func (b *SecretBuilderImpl) CleanTreeUpdates() {
	cleanTreeUpdates(b.GateTree.Secrets)
	cleanTreeUpdates(b.UnmanagedGateTree.Secrets)
}
