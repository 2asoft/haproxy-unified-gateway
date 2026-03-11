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
	"encoding/json"
	"log/slog"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
)

// A Global represents a Kubernetes Global CR
type Global struct {
	// K8sResource is the source resource.
	K8sResource *v3.Global
	// TreeStatus
	TreeStatus TreeUpdate[Global]
}

// NewGlobal creates a new Global for the GateTree.
func NewGlobal(k8sObject *v3.Global) *Global {
	return &Global{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[Global]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
	}
}

// SetAsUpserted marks the Global CR as upserted in the GateTree.
func (s *Global) SetAsUpserted(logger *slog.Logger, newK8sResource *v3.Global) {
	setResourceStatus(logger, s, newK8sResource, store.StatusUpserted)
	s.K8sResource = newK8sResource
}

// SetAsDeleted marks the Global as deleted in the GateTree.
func (s *Global) SetAsDeleted(logger *slog.Logger) {
	setResourceStatus(logger, s, nil, store.StatusDeleted)
	s.K8sResource = nil
}

// DeepCopy creates a deep copy of the Global.
func (s *Global) DeepCopy() *Global {
	if s == nil {
		return nil
	}
	// Save TreeStatus
	treeStatus := s.TreeStatus
	s.TreeStatus = TreeUpdate[Global]{}

	var copied Global
	data, err := json.Marshal(s) // Serialize to JSON
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct	return &copied
	// Restore TreeStatus
	s.TreeStatus = treeStatus
	return &copied
}

// GetTreeStatus returns the TreeStatus of the Global.
func (s *Global) GetTreeStatus() *TreeUpdate[Global] {
	return &s.TreeStatus
}

// SetTreeStatus sets the TreeStatus of the Global.
func (s *Global) SetTreeStatus(treeStatus TreeUpdate[Global]) {
	s.TreeStatus = treeStatus
}
