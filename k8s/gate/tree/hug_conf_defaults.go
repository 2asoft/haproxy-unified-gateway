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

// onDeletedSubsystemDefaults handles the deletion of a HugConf by marking the
// associated Defaults node in the GateTree as deleted, if one is present.
func (b *HugConfBuilderImpl) onDeletedSubsystemDefaults(_ store.Update[*v3.HugConf]) {
	if b.GateTree.Defaults != nil {
		b.GateTree.Defaults.SetAsDeleted(b.Logger)
	}
}

// onUpsertedSubsystemDefaults handles an upsert of a HugConf by resolving its
// DefaultsRef against the ClusterStore and updating the GateTree accordingly.
// If no DefaultsRef is configured or the referenced Defaults CR does not exist in
// the store, any existing Defaults node in the tree is marked as deleted.
// Otherwise the tree node is upserted with the current CR, or created if not
// yet present.
func (b *HugConfBuilderImpl) onUpsertedSubsystemDefaults(hugConfUpdate store.Update[*v3.HugConf]) {
	hugConf := hugConfUpdate.NewObject
	if hugConf == nil {
		return
	}

	// Get the Defaults from the ClusterStore
	defaultsRef := hugConf.Spec.DefaultsRef
	if defaultsRef == nil {
		// No defaults ref configured: treat as deleted in the Tree if present
		if b.GateTree.Defaults != nil {
			b.GateTree.Defaults.SetAsDeleted(b.Logger)
		}
		return
	}

	defaultsNsName := utils.GetNamespacedName(defaultsRef.Name, defaultsRef.Namespace, hugConf.Namespace)
	defaultsCR, exists := b.ClusterStore.DefaultsCRs[defaultsNsName]

	// If it does not exist in the ClusterStore
	// Set it as deleted in the Tree (if present)
	if !exists {
		if b.GateTree.Defaults != nil {
			b.GateTree.Defaults.SetAsDeleted(b.Logger)
		}
		return
	}

	// If it does exist in the ClusterStore
	// Set it as upserted in the Tree (if present), if not present add it to the Tree and set it as upserted.
	if b.GateTree.Defaults != nil {
		b.GateTree.Defaults.SetAsUpserted(b.Logger, defaultsCR)
	} else {
		b.GateTree.Defaults = NewDefaultsCR(defaultsCR)
	}
}
