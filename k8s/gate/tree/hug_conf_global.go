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
	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
)

// onDeletedSubsystemGlobal handles the deletion of a HugConf by marking the
// associated Global node in the GateTree as deleted, if one is present.
func (b *HugConfBuilderImpl) onDeletedSubsystemGlobal(_ store.Update[*v3.HugConf]) {
	if b.GateTree.Global != nil {
		b.GateTree.Global.SetAsDeleted(b.Logger)
	}
}

// onUpsertedSubsystemGlobal handles an upsert of a HugConf by resolving its
// GlobalRef against the ClusterStore and updating the GateTree accordingly.
// If no GlobalRef is configured or the referenced Global CR does not exist in
// the store, any existing Global node in the tree is marked as deleted.
// Otherwise the tree node is upserted with the current CR, or created if not
// yet present.
func (b *HugConfBuilderImpl) onUpsertedSubsystemGlobal(hugConfUpdate store.Update[*v3.HugConf]) {
	hugConf := hugConfUpdate.NewObject
	if hugConf == nil {
		return
	}

	// Get the Global from the ClusterStore
	globalRef := hugConf.Spec.GlobalRef
	if globalRef == nil {
		// No global ref configured: treat as deleted in the Tree if present
		if b.GateTree.Global != nil {
			b.GateTree.Global.SetAsDeleted(b.Logger)
		}
		return
	}

	globalNsName := utils.GetNamespacedName(globalRef.Name, globalRef.Namespace, hugConf.Namespace)
	globalCR, exists := b.ClusterStore.GlobalCRs[globalNsName]

	// If it does not exist in the ClusterStore
	// Set it as deleted in the Tree (if present)
	if !exists {
		if b.GateTree.Global != nil {
			b.GateTree.Global.SetAsDeleted(b.Logger)
		}
		return
	}

	// If it does exist in the ClusterStore
	// Set it as upserted in the Tree (if present), if not present add it to the Tree and set it as upserted.
	if b.GateTree.Global != nil {
		b.GateTree.Global.SetAsUpserted(b.Logger, globalCR)
	} else {
		b.GateTree.Global = NewGlobal(globalCR)
	}
}
