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
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	needsHugGatesReferencesRebuild := rm.needsReferencedHugGatesRebuild()
	needsGatewayClassesReferencesRebuild := rm.needsReferencedGatewayClassesRebuild()
	needsSecretsReferencesRebuild := rm.needsReferencedSecretsRebuild()
	needGatewaysReferencesRebuild := rm.needsReferencedGatewaysRebuild()

	if !needsHugGatesReferencesRebuild && !needsGatewayClassesReferencesRebuild && !needsSecretsReferencesRebuild {
		return
	}

	// HugGates refs
	if needsHugGatesReferencesRebuild {
		for _, gwc := range rm.ClusterStore.GatewayClasses {
			paramsRefKey, hasParamsRef := getGatewayClassParamsRefKey(gwc)
			if hasParamsRef {
				rm.ReferencedObjects.ReferencedHugGates.AddReferencedBy(rm.Logger, paramsRefKey, gwc)
			}
		}
	}

	// GatewayClass refs
	for _, gw := range rm.ClusterStore.Gateways {
		gwcKey := client.ObjectKey{Name: string(gw.Spec.GatewayClassName)}
		if needsGatewayClassesReferencesRebuild {
			rm.ReferencedObjects.ReferencedGatewayClasses.AddReferencedBy(rm.Logger, gwcKey, gw)
		}
		if needsHugGatesReferencesRebuild {
			// Direct HugGates Ref
			paramsRefKey, hasParamsRef := getGatewayParamsRefKey(gw)
			if hasParamsRef {
				rm.ReferencedObjects.ReferencedHugGates.AddReferencedBy(rm.Logger, paramsRefKey, gw)
			}

			// Now find the GatewayClass and their HugGates
			gwc, gwcOK := rm.ClusterStore.GatewayClasses[gwcKey]
			if gwcOK {
				paramsRefKey, hasParamsRef := getGatewayClassParamsRefKey(gwc)
				if hasParamsRef {
					rm.ReferencedObjects.ReferencedHugGates.AddReferencedBy(rm.Logger, paramsRefKey, gw)
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

	// Routes
	if needGatewaysReferencesRebuild {
		for _, route := range rm.ClusterStore.HTTPRoutes {
			routekey := client.ObjectKey{Namespace: route.Namespace, Name: route.Name}
			// TODO: fix this, do not assume its connected
			rm.ReferencedObjects.ReferencedGateway.AddReferencedBy(rm.Logger, routekey, route)
			// parentRefExists := len(route.Spec.ParentRefs) > 0
			// if parentRefExists {
			// find the gateway by name (if more than one, check by hostname)
			// } else {
			//  find the gateway by hostname
			// }
		}
	}
}

func (rm *ReferenceManager) needsReferencedHugGatesRebuild() bool {
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

func (rm *ReferenceManager) needsReferencedGatewaysRebuild() bool {
	return len(rm.ClusterStore.Updates.HTTPRoutes) > 0
}

func (rm *ReferenceManager) cleanReferencedObjects() {
	if rm.needsReferencedGatewayClassesRebuild() {
		rm.ReferencedObjects.ReferencedGatewayClasses.CleanOwners()
	}
	if rm.needsReferencedGatewaysRebuild() {
		rm.ReferencedObjects.ReferencedGateway.CleanOwners()
	}
	if rm.needsReferencedHugGatesRebuild() {
		rm.ReferencedObjects.ReferencedHugGates.CleanOwners()
	}
	rm.ReferencedObjects.PreviousReferencedSecrets = rm.ReferencedObjects.ReferencedSecrets.DeepCopy()
	if rm.needsReferencedSecretsRebuild() {
		rm.ReferencedObjects.ReferencedSecrets.CleanOwners()
	}
}
