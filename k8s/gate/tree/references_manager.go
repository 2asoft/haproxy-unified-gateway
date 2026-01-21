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
	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"
)

type ReferenceManager struct {
	ControllerStore
}

func NewReferenceManager(controllerStore ControllerStore) *ReferenceManager {
	return &ReferenceManager{
		ControllerStore: controllerStore,
	}
}

func (rm *ReferenceManager) UpdateRefences() {
	rm.cleanReferencedObjects()
	needsSecretsReferencesRebuild := rm.needsReferencedSecretsRebuild()

	if !needsSecretsReferencesRebuild {
		return
	}

	// Secrets refs
	rm.buildSecretReferences()
}

func (rm *ReferenceManager) needsReferencedSecretsRebuild() bool {
	return len(rm.ClusterStore.Updates.Gateways) > 0
}

func (rm *ReferenceManager) buildSecretReferences() {
	if needsSecretsReferencesRebuild := rm.needsReferencedSecretsRebuild(); !needsSecretsReferencesRebuild {
		return
	}

	for _, gw := range rm.ClusterStore.Gateways {
		for _, listener := range gw.Spec.Listeners {
			if listener.TLS == nil {
				continue
			}
			for _, certRef := range listener.TLS.CertificateRefs {
				// We only accept v1.Secret
				if !isSecretGroupKindSupported(certRef) {
					continue
				}
				nsName := GetCertificateRefNamespacedName(certRef, gw)
				ownerGVK := rm.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway)
				rm.ReferencedObjects.ReferencedSecrets.AddReferencedByUsingKeys(rm.Logger, nsName, ListenerKey(gw, listener), ownerGVK)
			}
		}
	}
}

func (rm *ReferenceManager) cleanReferencedObjects() {
	rm.ReferencedObjects.PreviousReferencedSecrets = rm.ReferencedObjects.ReferencedSecrets.DeepCopy()
	if rm.needsReferencedSecretsRebuild() {
		rm.ReferencedObjects.ReferencedSecrets.CleanOwners()
	}
}
