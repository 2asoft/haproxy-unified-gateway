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
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage"
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ Builder = &GatewayBuilderImpl{}

type GatewayBuilderImpl struct {
	certStorage                       storage.CertificateStorage
	portsWithOneListener              map[gatewayv1.PortNumber]struct{}
	portsWithMutipleListeners         map[gatewayv1.PortNumber][]gatewaylistener
	previousPortsWithMutipleListeners map[gatewayv1.PortNumber][]gatewaylistener
	ControllerStore
}

type gatewaylistener struct {
	listenerKey client.ObjectKey
	gatewayKey  client.ObjectKey
}

type GatewayBuilderParams struct {
	storage.CertificateStorage
	ControllerStore
}

func NewGatewayBuilder(params GatewayBuilderParams) Builder {
	return &GatewayBuilderImpl{
		ControllerStore: params.ControllerStore,
		certStorage:     params.CertificateStorage,
	}
}

// --------------------
// GateTree Updates
// --------------------

func (b *GatewayBuilderImpl) ComputeTreeUpdates() {
	b.resetListenerConflicts()
	b.addIndirectClusterStoreUpdates()

	// After this step, the clusterStore.Updates contains all impacted Gateways
	// Including the one impacted by:
	// - HugGate updates
	// - GatewayClass updates
	b.computeGateTreeUpdates()
	// Here we check for Listener conflicts
}

func (b *GatewayBuilderImpl) addIndirectClusterStoreUpdates() {
	// Indirect from HugGate
	b.addIndirectGatewaysFromHugGates()
	// Indirect from GatewayClass
	b.addIndirectGatewaysFromGatewayClasses()
	// Indirect from Secret
	b.addIndirectGatewaysFromSecrets()
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromHugGates() {
	for _, hugGateUpdate := range b.ClusterStore.Updates.HugGates {
		b.addIndirectGatewaysFromHugGate(hugGateUpdate)
	}
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromHugGate(hugGateUpdate store.Update[*v3.HugGate]) {
	addIndirectFromReferenced(
		hugGateUpdate,
		b.ControllerStore.ReferencedObjects.ReferencedHugGates,
		b.ControllerStore.ClusterStore.Gateways,
		b.ClusterStore.Updates.Gateways,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway),
		nil, // no ownerKey transformation
	)
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromGatewayClasses() {
	for _, gwcUpdate := range b.ClusterStore.Updates.GatewayClasses {
		b.addIndirectGatewaysFromGatewayClass(gwcUpdate)
	}
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromGatewayClass(gwcUpdate store.Update[*gatewayv1.GatewayClass]) {
	addIndirectFromReferenced(
		gwcUpdate,
		b.ReferencedObjects.ReferencedGatewayClasses,
		b.ClusterStore.Gateways,
		b.ClusterStore.Updates.Gateways,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway),
		nil, // no ownerKey transformation
	)
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromSecrets() {
	for _, secretUpdate := range b.ClusterStore.Updates.Secrets {
		b.addIndirectGatewaysFromSecret(secretUpdate)
	}
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromSecret(secretUpdate store.Update[*v1.Secret]) {
	addIndirectFromReferenced(
		secretUpdate,
		b.ReferencedObjects.ReferencedSecrets,
		b.ClusterStore.Gateways,
		b.ClusterStore.Updates.Gateways,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway),
		ConvertListenerKeyToGatewayKey,
	)
}

func (b *GatewayBuilderImpl) computeGateTreeUpdates() {
	// Compute the upserted/delete Tree Gateways
	for gwKey, gwUpdate := range b.ClusterStore.Updates.Gateways {
		b.computeTreeGatewayUpdate(gwKey, gwUpdate)
	}

	// Check for conflicts
	// Add all Gateways that have a conflict in the list of updated Gateways
	// in order to recompute the checks and status (both Gateway and listeners)
	b.checkListenerConflicts()

	for _, treeGw := range b.ControllerStore.GateTree.Gateways {
		if treeGw.TreeStatus.Status != store.StatusUpserted {
			continue
		}

		if treeGw.isManaged() {
			// Process Listeners
			b.buildListeners(treeGw)

			// Compute status only if managed Gateway
			// If not managed, then we should not update the status
			// treeGw.BuildManagementConditions()
			// Build Listener conditions
			for _, listener := range treeGw.Listeners {
				listener.BuildConditions(treeGw)
			}
			treeGw.checkListenerConflicts(b.portsWithOneListener)
			treeGw.BuildConditions()
		}
	}
}

func (b *GatewayBuilderImpl) computeTreeGatewayUpdate(gwKey client.ObjectKey, gwUpdate store.Update[*gatewayv1.Gateway]) {
	var treeGw *Gateway
	alreadyManagedTreeGw, alreadyManagedTreeGwOK := b.GateTree.Gateways[gwKey]
	alreadyUnmanagedTreeGw, alreadyUnmanagedTreeGwOK := b.UnmanagedGateTree.Gateways[gwKey]

	if alreadyManagedTreeGwOK {
		treeGw = alreadyManagedTreeGw
	} else if alreadyUnmanagedTreeGwOK {
		treeGw = alreadyUnmanagedTreeGw
	}

	switch gwUpdate.Status {
	case store.StatusUpserted:
		if treeGw != nil {
			treeGw.SetAsUpserted(b.Logger, gwUpdate.NewObject)
		} else {
			treeGw = NewGateway(gwUpdate.NewObject)
		}

		// Do we keep it in Managed or Unmanaged???
		b.processManagementChecks(treeGw)

	case store.StatusDeleted:
		if treeGw != nil {
			treeGw.SetAsDeleted(b.Logger)
		}
		// else nothing to do
		// It did not exists, it's deleted, noop
	}
}

func (b *GatewayBuilderImpl) processManagementChecks(treeGw *Gateway) {
	// If the GatewayClass is not in the store, it means that the GatewayClass is not managed by our controller
	if ok := b.ControllerStore.CheckGatewayClassExists(string(treeGw.K8sResource.Spec.GatewayClassName)); !ok {
		return
	}

	treeGw.checkParametersRef(b.ControllerStore)
	treeGw.checkGatewayClassIsValid(b.ControllerStore)
	treeGw.Valid = treeGw.CheckParamsRef.Valid && treeGw.CheckValidGatewayClass.Valid

	if treeGw.isManaged() {
		treeGw.SetAsManaged(b.Logger, b.ControllerStore)
	} else {
		treeGw.SetAsUnmanaged(b.Logger, b.ControllerStore)
	}
}

// -----------------------------------------------

func (b *GatewayBuilderImpl) CleanTreeUpdates() {
	for gwKey, treeGw := range b.GateTree.Gateways {
		if treeGw.TreeStatus.Status == store.StatusDeleted {
			delete(b.GateTree.Gateways, gwKey)
			continue
		}
		treeGw.TreeStatus = TreeUpdate[Gateway]{}
	}
	for gwKey, treeGw := range b.UnmanagedGateTree.Gateways {
		if treeGw.TreeStatus.Status == store.StatusDeleted {
			delete(b.GateTree.Gateways, gwKey)
			continue
		}
		treeGw.TreeStatus = TreeUpdate[Gateway]{}
	}
}

func (b *GatewayBuilderImpl) buildListeners(treeGw *Gateway) {
	processedListeners := make(map[string]*Listener)

	for _, listener := range treeGw.K8sResource.Spec.Listeners {
		kinds := supportedKinds(listener, gateSupportedRouteKindsByProtocol)
		processedListener := Listener{
			owner:             client.ObjectKeyFromObject(treeGw.K8sResource),
			K8sResource:       listener,
			AllowedRouteKinds: kinds,
			Conditions:        make(generic.Conditions),
		}
		processedListeners[string(listener.Name)] = &processedListener
	}
	treeGw.Listeners = processedListeners

	// Performs all needed checks
	for _, listener := range treeGw.Listeners {
		listener.resetChecks()
		switch listener.K8sResource.Protocol {
		// This switch will be completed with all needed checks per protocol
		case gatewayv1.HTTPProtocolType:
			listener.checkRouteGroupKind(treeGw, gateSupportedRouteKindsByProtocol)
			listener.checkProtocol(gateSupportedRouteKindsByProtocol)
			listener.checkConflict(treeGw, b.portsWithMutipleListeners)
		case gatewayv1.HTTPSProtocolType:
			listener.checkRouteGroupKind(treeGw, gateSupportedRouteKindsByProtocol)
			listener.checkCertificateRefs(treeGw, b.GateTree.Secrets)
			listener.checkProtocol(gateSupportedRouteKindsByProtocol)
			listener.checkConflict(treeGw, b.portsWithMutipleListeners)
		default:
			listener.checkProtocol(gateSupportedRouteKindsByProtocol)
			listener.checkConflict(treeGw, b.portsWithMutipleListeners)
		}
	}
}

func (b *GatewayBuilderImpl) resetListenerConflicts() {
	b.previousPortsWithMutipleListeners = b.portsWithMutipleListeners
	b.portsWithOneListener = make(map[gatewayv1.PortNumber]struct{})
	b.portsWithMutipleListeners = make(map[gatewayv1.PortNumber][]gatewaylistener)
}

// -----------------------------------------------

// checkListenerConflicts checks the conflicts between all Gateway listeners
// We accept only 1 Gateway Listener per port
// For now, as there are only a few number of Gateways, we do this check on all Gateway/ all listeners
func (b *GatewayBuilderImpl) checkListenerConflicts() {
	oldGwWithPortConflicts := b.previousGatewaysWithPortConflicts()
	listenersByPort := make(map[gatewayv1.PortNumber][]gatewaylistener)

	for _, treeGw := range b.GateTree.Gateways {
		if treeGw.K8sResource == nil {
			// ... deleted
			continue
		}
		for _, listener := range treeGw.K8sResource.Spec.Listeners {
			if _, ok := listenersByPort[listener.Port]; !ok {
				listenersByPort[listener.Port] = []gatewaylistener{}
			}
			gl := gatewaylistener{
				listenerKey: ListenerKey(treeGw.K8sResource, listener),
				gatewayKey:  client.ObjectKeyFromObject(treeGw.K8sResource),
			}

			listenersByPort[listener.Port] = append(listenersByPort[listener.Port], gl)
		}
	}

	for port, gls := range listenersByPort {
		if len(gls) > 1 {
			b.portsWithMutipleListeners[port] = gls
			continue
		}
		b.portsWithOneListener[port] = struct{}{}
	}
	newGwWithPortConflict := b.gatewaysWithPortConflicts()
	oldAndNewGwWithPortConflicts := map[client.ObjectKey]struct{}{}
	for gwKey := range newGwWithPortConflict {
		oldAndNewGwWithPortConflicts[gwKey] = struct{}{}
	}
	for gwKey := range oldGwWithPortConflicts {
		oldAndNewGwWithPortConflicts[gwKey] = struct{}{}
	}
	for gwKey := range oldAndNewGwWithPortConflicts {
		treeGw := b.ControllerStore.GateTree.Gateways[gwKey]
		// Set the treeGw as UPSERTED
		if treeGw.TreeStatus.Status != store.StatusUpserted && treeGw.TreeStatus.Status != store.StatusDeleted {
			treeGw.SetAsUpserted(b.Logger, treeGw.K8sResource)
			b.processManagementChecks(treeGw)
		}
	}
}

// gatewaysWithPortConflicts returns a map of Gateway keys which have a conflict
func (b *GatewayBuilderImpl) gatewaysWithPortConflicts() map[client.ObjectKey]struct{} {
	return gatewaysWithPortConflicts(b.portsWithMutipleListeners)
}

// previousGatewaysWithPortConflicts returns a map of Gateway keys which have a conflict
func (b *GatewayBuilderImpl) previousGatewaysWithPortConflicts() map[client.ObjectKey]struct{} {
	return gatewaysWithPortConflicts(b.previousPortsWithMutipleListeners)
}

func gatewaysWithPortConflicts(portsWithMultipleListeners map[gatewayv1.PortNumber][]gatewaylistener) map[client.ObjectKey]struct{} {
	gwKeys := map[client.ObjectKey]struct{}{}
	for _, gls := range portsWithMultipleListeners {
		for _, gl := range gls {
			gwKeys[gl.gatewayKey] = struct{}{}
		}
	}
	return gwKeys
}
