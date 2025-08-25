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
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// A Secret represents a Kubernetes Secret
type Secret struct {
	// K8sResource is the source resource.
	K8sResource *v1.Secret
	// TreeStatus
	TreeStatus TreeUpdate[Secret]
}

func NewSecret(k8sObject *v1.Secret) *Secret {
	return &Secret{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[Secret]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
	}
}

func (s *Secret) SetAsUpserted(logger *slog.Logger, newK8sResource *v1.Secret) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Upserted",
		logging.LogAttrObjectKey(newK8sResource))
	s.TreeStatus.Status = store.StatusUpserted
	s.TreeStatus.OldTreeResource = s.DeepCopy()
	s.K8sResource = newK8sResource
}

func (s *Secret) SetAsDeleted(logger *slog.Logger) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Deleted",
		logging.LogAttrObjectKey(s.K8sResource))
	s.TreeStatus.Status = store.StatusDeleted
	s.TreeStatus.OldTreeResource = s.DeepCopy()
	s.K8sResource = nil
}

func (s *Secret) SetAsManaged(logger *slog.Logger, cs ControllerStore) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Managed",
		logging.LogAttrObjectKey(s.K8sResource))
	key := client.ObjectKeyFromObject(s.K8sResource)
	cs.GateTree.Secrets[key] = s
}

func (s *Secret) DeepCopy() *Secret {
	if s == nil {
		return nil
	}
	// Save TreeStatus
	treeStatus := s.TreeStatus
	s.TreeStatus = TreeUpdate[Secret]{}

	var copied Secret
	data, err := json.Marshal(s) // Serialize to JSON
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct	return &copied
	// Restore TreeStatus
	s.TreeStatus = treeStatus
	return &copied
}

func getNamespacedName(certRef gatewayv1.SecretObjectReference, gw *gatewayv1.Gateway) types.NamespacedName {
	namespace := gw.Namespace
	if certRef.Namespace != nil {
		namespace = string(*certRef.Namespace)
	}
	return types.NamespacedName{
		Namespace: namespace,
		Name:      string(certRef.Name),
	}
}
