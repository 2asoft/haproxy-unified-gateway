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
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
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
	needGatewaysReferencesRebuild := rm.needsReferencedGatewaysRebuild()
	needsSecretsReferencesRebuild := rm.needsReferencedSecretsRebuild()
	needServicesReferencesRebuild := rm.needsReferencedServicesRebuild()

	if !needsHugGatesReferencesRebuild && !needsGatewayClassesReferencesRebuild &&
		!needsSecretsReferencesRebuild && !needGatewaysReferencesRebuild && !needServicesReferencesRebuild {
		return
	}

	// HugGates refs
	rm.buildHugGatesReferences()

	// GatewayClass refs
	rm.buildGatewayClassReferences()

	// Secrets refs
	rm.buildSecretReferences()

	// Gateways refs
	rm.buildGatewayReferences()

	// Services refs
	rm.buildServiceReferences()

	// BackendCR refs
	rm.buildBackendCRReferences()
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

func (rm *ReferenceManager) needsReferencedServicesRebuild() bool {
	return len(rm.ClusterStore.Updates.Services) > 0
}

func (rm *ReferenceManager) needsReferencedBackendCRsRebuild() bool {
	return len(rm.ClusterStore.Updates.HTTPRoutes) > 0
}

func (rm *ReferenceManager) buildHugGatesReferences() {
	if needsHugGatesReferencesRebuild := rm.needsReferencedHugGatesRebuild(); !needsHugGatesReferencesRebuild {
		return
	}

	for _, gwc := range rm.ClusterStore.GatewayClasses {
		paramsRefKey, hasParamsRef := getGatewayClassParamsRefKey(gwc)
		if hasParamsRef {
			rm.ReferencedObjects.ReferencedHugGates.AddReferencedBy(rm.Logger, paramsRefKey, gwc)
		}
	}
}

func (rm *ReferenceManager) buildGatewayClassReferences() {
	needsHugGatesReferencesRebuild := rm.needsReferencedHugGatesRebuild()
	needsGatewayClassesReferencesRebuild := rm.needsReferencedGatewayClassesRebuild()

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

func (rm *ReferenceManager) buildGatewayReferences() {
	if needGatewaysReferencesRebuild := rm.needsReferencedGatewaysRebuild(); !needGatewaysReferencesRebuild {
		return
	}

	for _, route := range rm.ClusterStore.HTTPRoutes {
		for _, parentRef := range route.Spec.ParentRefs {
			// We only accept v1.Gateway
			if !isParentRefGroupKindSupported(parentRef, rm.ControllerStore.ExtractGVK) {
				continue
			}
			nsName := GetParentRefNamespacedName(parentRef, route)
			rm.ReferencedObjects.ReferencedGateways.AddReferencedBy(rm.Logger, nsName, route)
		}
	}
}

func (rm *ReferenceManager) buildServiceReferences() {
	if needServicesReferencesRebuild := rm.needsReferencedServicesRebuild(); !needServicesReferencesRebuild {
		return
	}

	for _, route := range rm.ClusterStore.HTTPRoutes {
		for _, rule := range route.Spec.Rules {
			for _, backendRef := range rule.BackendRefs {
				// We only accept v1.Service
				if !isBackendRefGroupKindSupported(backendRef.BackendObjectReference, rm.ControllerStore.ExtractGVK) {
					continue
				}
				nsName := GetBackendRefNamespacedName(backendRef.BackendObjectReference, route)
				rm.ReferencedObjects.ReferencedServices.AddReferencedBy(rm.Logger, nsName, route)
			}
		}
	}
}

func (rm *ReferenceManager) buildBackendCRReferences() {
	if !rm.needsReferencedBackendCRsRebuild() {
		return
	}

	for _, route := range rm.ClusterStore.HTTPRoutes {
		for _, rule := range route.Spec.Rules {
			// BackendRef Filters
			for _, backendRef := range rule.BackendRefs {
				for _, filter := range backendRef.Filters {
					if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef {
						continue
					}
					// We only accept v3.Backend
					if !IsFilterExtensionRefKindSupported(filter.ExtensionRef, rm.ControllerStore.ExtractGVK) {
						continue
					}
					nsName := types.NamespacedName{
						Namespace: route.Namespace,
						Name:      string(filter.ExtensionRef.Name),
					}
					rm.ReferencedObjects.ReferencedBackendCRs.AddReferencedBy(rm.Logger, nsName, route)
				}
			}
		}
	}
}

func (rm *ReferenceManager) cleanReferencedObjects() {
	if rm.needsReferencedGatewayClassesRebuild() {
		rm.ReferencedObjects.ReferencedGatewayClasses.CleanOwners()
	}
	if rm.needsReferencedHugGatesRebuild() {
		rm.ReferencedObjects.ReferencedHugGates.CleanOwners()
	}
	rm.ReferencedObjects.PreviousReferencedSecrets = rm.ReferencedObjects.ReferencedSecrets.DeepCopy()
	if rm.needsReferencedSecretsRebuild() {
		rm.ReferencedObjects.ReferencedSecrets.CleanOwners()
	}
	if rm.needsReferencedGatewaysRebuild() {
		rm.ReferencedObjects.ReferencedGateways.CleanOwners()
	}
	if rm.needsReferencedServicesRebuild() {
		rm.ReferencedObjects.ReferencedServices.CleanOwners()
	}

	if rm.needsReferencedBackendCRsRebuild() {
		rm.ReferencedObjects.ReferencedBackendCRs.CleanOwners()
	}
}
