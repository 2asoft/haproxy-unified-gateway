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

// A DefaultsCR represents a Kubernetes Defaults CR
type DefaultsCR struct {
	// K8sResource is the source resource.
	K8sResource *v3.Defaults
	// TreeStatus
	TreeStatus TreeUpdate[DefaultsCR]
}

// NewDefaultsCR creates a new DefaultsCR for the GateTree.
func NewDefaultsCR(k8sObject *v3.Defaults) *DefaultsCR {
	return &DefaultsCR{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[DefaultsCR]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
	}
}

// SetAsUpserted marks the Defaults CR as upserted in the GateTree.
func (s *DefaultsCR) SetAsUpserted(logger *slog.Logger, newK8sResource *v3.Defaults) {
	setResourceStatus(logger, s, newK8sResource, store.StatusUpserted)
	s.K8sResource = newK8sResource
}

// SetAsDeleted marks the Defaults CR as deleted in the GateTree.
func (s *DefaultsCR) SetAsDeleted(logger *slog.Logger) {
	setResourceStatus(logger, s, nil, store.StatusDeleted)
	s.K8sResource = nil
}

// DeepCopy creates a deep copy of the DefaultsCR.
func (s *DefaultsCR) DeepCopy() *DefaultsCR {
	if s == nil {
		return nil
	}
	// Save TreeStatus
	treeStatus := s.TreeStatus
	s.TreeStatus = TreeUpdate[DefaultsCR]{}

	var copied DefaultsCR
	data, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(data, &copied)
	// Restore TreeStatus
	s.TreeStatus = treeStatus
	return &copied
}

// GetTreeStatus returns the TreeStatus of the DefaultsCR.
func (s *DefaultsCR) GetTreeStatus() *TreeUpdate[DefaultsCR] {
	return &s.TreeStatus
}

// SetTreeStatus sets the TreeStatus of the DefaultsCR.
func (s *DefaultsCR) SetTreeStatus(treeStatus TreeUpdate[DefaultsCR]) {
	s.TreeStatus = treeStatus
}
