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
package haproxy

import (
	"encoding/json"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
)

// processGlobal reads the Global node from the GateTree and reconciles it into
// the Configuration. If the node is upserted, the corresponding models.Global
// is written to the structured store and recorded in HaproxyConfDiffs (Created
// or Updated). If the node is deleted, it is removed from the structured store
// and recorded as Deleted. A nil or unchanged node is a no-op.
func (b *HaproxyConfMgrImpl) processGlobal() error {
	treeGlobal := b.controllerStore.GateTree.Global
	if treeGlobal == nil {
		return nil
	}

	switch treeGlobal.TreeStatus.Status {
	case store.StatusUpserted:
		if treeGlobal.K8sResource == nil {
			return nil
		}
		global := treeGlobal.K8sResource.Spec.Global
		return b.configuration.upsertGlobal(b.logger, &global, treeGlobal.K8sResource.Spec.MergeStrategy)
	case store.StatusDeleted:
		return b.configuration.deleteGlobal(b.logger)
	}
	return nil
}

// DeepCopyGlobal returns a deep copy of the given models.Global using JSON
// serialisation. Returns nil, nil when original is nil.
func DeepCopyGlobal(original *models.Global) (*models.Global, error) {
	if original == nil {
		return nil, nil
	}
	var copied models.Global
	data, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(data, &copied)
	return &copied, nil
}
