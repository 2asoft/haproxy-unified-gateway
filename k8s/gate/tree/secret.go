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
	"strings"

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

// NewSecret creates a new Secret for the GateTree.
func NewSecret(k8sObject *v1.Secret) *Secret {
	return &Secret{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[Secret]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
	}
}

// SetAsUpserted marks the Secret as upserted in the GateTree.
func (s *Secret) SetAsUpserted(logger *slog.Logger, newK8sResource *v1.Secret) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Upserted",
		logging.LogAttrObjectKey(newK8sResource))
	s.TreeStatus.Status = store.StatusUpserted
	s.TreeStatus.OldTreeResource = s.DeepCopy()
	s.K8sResource = newK8sResource
}

// SetAsDeleted marks the Secret as deleted in the GateTree.
func (s *Secret) SetAsDeleted(logger *slog.Logger) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Deleted",
		logging.LogAttrObjectKey(s.K8sResource))
	s.TreeStatus.Status = store.StatusDeleted
	s.TreeStatus.OldTreeResource = s.DeepCopy()
	s.K8sResource = nil
}

// SetAsManaged moves the Secret to the managed GateTree.
func (s *Secret) SetAsManaged(logger *slog.Logger, cs ControllerStore) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Managed",
		logging.LogAttrObjectKey(s.K8sResource))
	key := client.ObjectKeyFromObject(s.K8sResource)
	cs.GateTree.Secrets[key] = s
}

// DeepCopy creates a deep copy of the Secret.
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

// GetCertificateRefNamespacedName returns the namespaced name for a certificate reference,
// using the Gateway's namespace as a default if the reference does not specify one.
func GetCertificateRefNamespacedName(certRef gatewayv1.SecretObjectReference, gw *gatewayv1.Gateway) types.NamespacedName {
	namespace := gw.Namespace
	if certRef.Namespace != nil {
		namespace = string(*certRef.Namespace)
	}
	return types.NamespacedName{
		Namespace: namespace,
		Name:      string(certRef.Name),
	}
}

// ListenerKey returns the Certificate owner key appending the listener name to it
// For Gateway ns/gateway, if the Listener name is "https", will return
// ns/gateway_https
// = Listener Key
func ListenerKey(gw *gatewayv1.Gateway, listener gatewayv1.Listener) client.ObjectKey {
	return client.ObjectKey{
		Namespace: gw.Namespace,
		Name: fmt.Sprintf("%s_%s",
			gw.Name,
			listener.Name,
		),
	}
}

// ConvertListenerKeyToGatewayKey converts a listener key back to a gateway key.
// It assumes the listener key is in the format "gateway-name_listener-name", built by the previous ListenerKey function.
// For a listener key with namespace "ns" and name "my-gateway_https",
// it returns a gateway key with namespace "ns" and name "my-gateway".
func ConvertListenerKeyToGatewayKey(listenerKey client.ObjectKey) client.ObjectKey {
	gatewayKey, _, err := ConvertListenerKeyToGatewayKeyAndListenerName(listenerKey)
	if err != nil {
		return listenerKey
	}
	return gatewayKey
}

// ConvertListenerKeyToGatewayKeyAndListenerName converts a listener key back to a gateway key and listener name.
// It assumes the listener key is in the format "gateway-name_listener-name", built by the ListenerKey function.
// For a listener key with namespace "ns" and name "my-gateway_https",
// it returns a gateway key with namespace "ns" and name "my-gateway", the listener name "https", and no error.
// If the format is invalid, it returns an error.
func ConvertListenerKeyToGatewayKeyAndListenerName(listenerKey client.ObjectKey) (client.ObjectKey, string, error) {
	parts := strings.Split(listenerKey.Name, "_")
	if len(parts) != 2 {
		return client.ObjectKey{}, "", fmt.Errorf("invalid listener key format: %s", listenerKey.Name)
	}
	gatewayKey := client.ObjectKey{
		Namespace: listenerKey.Namespace,
		Name:      parts[0],
	}
	return gatewayKey, parts[1], nil
}

// isSecretGroupKindSupported checks if the provided certificate reference has a supported Group and Kind.
// It only supports core `v1.Secret` resources.
func isSecretGroupKindSupported(certRef gatewayv1.SecretObjectReference) bool {
	if certRef.Kind != nil && *certRef.Kind != "Secret" {
		return false
	}
	if certRef.Group != nil && *certRef.Group != "" {
		return false
	}
	return true
}
