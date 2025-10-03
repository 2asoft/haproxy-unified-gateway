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
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// A Service represents a Kubernetes Service
type Service struct {
	// K8sResource is the source resource.
	K8sResource *v1.Service
	// TreeStatus
	TreeStatus TreeUpdate[Service]
}

// NewService creates a new Service for the GateTree.
func NewService(k8sObject *v1.Service) *Service {
	return &Service{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[Service]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
	}
}

// SetAsUpserted marks the Service as upserted in the GateTree.
func (s *Service) SetAsUpserted(logger *slog.Logger, newK8sResource *v1.Service) {
	setResourceStatus(logger, s, newK8sResource, store.StatusUpserted)
	s.K8sResource = newK8sResource
}

// SetAsDeleted marks the Service as deleted in the GateTree.
func (s *Service) SetAsDeleted(logger *slog.Logger) {
	setResourceStatus(logger, s, nil, store.StatusDeleted)
	s.K8sResource = nil
}

// SetAsManaged moves the Service to the managed GateTree.
func (s *Service) SetAsManaged(logger *slog.Logger, cs ControllerStore) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprintf("%T MANAGED", *s),
		logging.LogAttrObjectKey(s.K8sResource))
	key := client.ObjectKeyFromObject(s.K8sResource)
	cs.GateTree.Services[key] = s
}

// DeepCopy creates a deep copy of the Service.
func (s *Service) DeepCopy() *Service {
	if s == nil {
		return nil
	}
	// Save TreeStatus
	treeStatus := s.TreeStatus
	s.TreeStatus = TreeUpdate[Service]{}

	var copied Service
	data, err := json.Marshal(s) // Serialize to JSON
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct	return &copied
	// Restore TreeStatus
	s.TreeStatus = treeStatus
	return &copied
}

// GetTreeStatus returns the TreeStatus of the Service.
func (s *Service) GetTreeStatus() *TreeUpdate[Service] {
	return &s.TreeStatus
}

// SetTreeStatus sets the TreeStatus of the Service.
func (s *Service) SetTreeStatus(treeStatus TreeUpdate[Service]) {
	s.TreeStatus = treeStatus
}
