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

import "sigs.k8s.io/controller-runtime/pkg/client"

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

	needsHaproGatesReferencesRebuild := rm.needsReferencedHaproxyGatesRebuild()
	needsGatewayClassesReferencesRebuild := rm.needsReferencedGatewayClassesRebuild()
	needsSecretsReferencesRebuild := rm.needsReferencedSecretsRebuild()

	if !needsHaproGatesReferencesRebuild && !needsGatewayClassesReferencesRebuild && !needsSecretsReferencesRebuild {
		return
	}

	// HAProxyGates refs
	if needsHaproGatesReferencesRebuild {
		for _, gwc := range rm.ClusterStore.GatewayClasses {
			paramsRefKey, hasParamsRef := getGatewayClassParamsRefKey(gwc)
			if hasParamsRef {
				rm.ReferencedObjects.ReferencedHaproxyGates.AddReferencedBy(rm.Logger, paramsRefKey, gwc)
			}
		}
	}

	// GatewayClass refs
	for _, gw := range rm.ClusterStore.Gateways {
		gwcKey := client.ObjectKey{Name: string(gw.Spec.GatewayClassName)}
		if needsGatewayClassesReferencesRebuild {
			rm.ReferencedObjects.ReferencedGatewayClasses.AddReferencedBy(rm.Logger, gwcKey, gw)
		}
		if needsHaproGatesReferencesRebuild {
			// Direct HaproxyGates Ref
			paramsRefKey, hasParamsRef := getGatewayParamsRefKey(gw)
			if hasParamsRef {
				rm.ReferencedObjects.ReferencedHaproxyGates.AddReferencedBy(rm.Logger, paramsRefKey, gw)
			}

			// Now find the GatewayClass and their HaproxyGates
			gwc, gwcOK := rm.ClusterStore.GatewayClasses[gwcKey]
			if gwcOK {
				paramsRefKey, hasParamsRef := getGatewayClassParamsRefKey(gwc)
				if hasParamsRef {
					rm.ReferencedObjects.ReferencedHaproxyGates.AddReferencedBy(rm.Logger, paramsRefKey, gw)
				}
			}
		}
	}

	// Secrets refs
	if needsSecretsReferencesRebuild {
		for _, gw := range rm.ClusterStore.Gateways {
			for _, listener := range gw.Spec.Listeners {
				if listener.TLS == nil {
					continue
				}
				for _, certRef := range listener.TLS.CertificateRefs {
					// We only accept v1.Secret
					if certRef.Kind != nil && *certRef.Kind != "Secret" {
						continue
					}
					if certRef.Group != nil && *certRef.Group != "" {
						continue
					}
					nsName := getNamespacedName(certRef, gw)
					rm.ReferencedObjects.ReferencedSecrets.AddReferencedBy(rm.Logger, nsName, gw)
				}
			}
		}
	}
}

func (rm *ReferenceManager) needsReferencedHaproxyGatesRebuild() bool {
	if len(rm.ClusterStore.Updates.GatewayClasses) > 0 || len(rm.ClusterStore.Updates.Gateways) > 0 {
		return true
	}
	return false
}

func (rm *ReferenceManager) needsReferencedGatewayClassesRebuild() bool {
	return len(rm.ClusterStore.Updates.Gateways) > 0
}

func (rm *ReferenceManager) needsReferencedSecretsRebuild() bool {
	return len(rm.ClusterStore.Updates.Gateways) > 0
}

func (rm *ReferenceManager) cleanReferencedObjects() {
	if rm.needsReferencedGatewayClassesRebuild() {
		rm.ReferencedObjects.ReferencedGatewayClasses.CleanOwners()
	}
	if rm.needsReferencedHaproxyGatesRebuild() {
		rm.ReferencedObjects.ReferencedHaproxyGates.CleanOwners()
	}
	if rm.needsReferencedSecretsRebuild() {
		rm.ReferencedObjects.ReferencedSecrets.CleanOwners()
	}
}
