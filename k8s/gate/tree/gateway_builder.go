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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ Builder = &GatewayBuilderImpl{}

type GatewayBuilderImpl struct {
	ControllerStore
}

type GatewayBuilderParams struct {
	ControllerStore
}

func NewGatewayBuilder(params GatewayBuilderParams) *GatewayBuilderImpl {
	return &GatewayBuilderImpl{
		ControllerStore: params.ControllerStore,
	}
}

// --------------------
// GateTree Updates
// --------------------

func (b *GatewayBuilderImpl) ComputeTreeUpdates() {
	b.addIndirectClusterStoreUpdates()
	// After this step, the clusterStore.Updates contains all impacted Gateways
	// Including the one impacted by:
	// - HaproxyGate updates
	// - GatewayClass updates
	b.computeGateTreeUpdates()
}

func (b *GatewayBuilderImpl) addIndirectClusterStoreUpdates() {
	// Indirect from HaproxyGate
	b.addIndirectGatewaysFromHaproxyGates()
	// Indirect from Gateway updated
	b.addIndirectGatewaysFromGatewayClasses()
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromHaproxyGates() {
	for _, haproxyGateUpdate := range b.ClusterStore.Updates.HaproxyGates {
		b.addIndirectGatewaysFromHaproxyGate(haproxyGateUpdate)
	}
}

func (b *GatewayBuilderImpl) addIndirectGatewaysFromHaproxyGate(haproxyGateUpdate store.Update[*v3.HaproxyGate]) {
	addIndirectFromReferenced(
		haproxyGateUpdate,
		b.ControllerStore.ReferencedObjects.ReferencedHaproxyGates,
		b.ControllerStore.ClusterStore.Gateways,
		b.ClusterStore.Updates.Gateways,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeGateway),
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
	)
}

func (b *GatewayBuilderImpl) computeGateTreeUpdates() {
	for gwKey, gwUpdate := range b.ClusterStore.Updates.Gateways {
		b.computeTreeGatewayUpdate(gwKey, gwUpdate)
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
		// If the GatewayClass is not in the store, it means that the GatewayClass is not managed by our controller
		if ok := treeGw.checkGatewayClassExistsInControllerStore(b.ControllerStore); !ok {
			return
		}

		// Do we keep it in Managed or Unmanaged???
		treeGw.processChecks(b.ControllerStore)

		if treeGw.isManaged() {
			treeGw.SetAsManaged(b.Logger, b.ControllerStore)
		} else {
			treeGw.SetAsUnmanaged(b.Logger, b.ControllerStore)
		}

		// Process Listeners
		b.buildListeners(treeGw)

		// Compute status only if managed Gateway
		// If not managed, then we should not update the status
		treeGw.BuildConditions()
		// Build Listener conditions
		for _, listener := range treeGw.Listeners {
			listener.BuildConditions(treeGw)
		}

	case store.StatusDeleted:
		if treeGw != nil {
			treeGw.SetAsDeleted(b.Logger)
		}
		// else nothing to do
		// It did not exists, it's deleted, noop
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

func (*GatewayBuilderImpl) buildListeners(treeGw *Gateway) {
	processedListeners := make([]*Listener, 0, len(treeGw.K8sResource.Spec.Listeners))

	for _, listener := range treeGw.K8sResource.Spec.Listeners {
		kinds := supportedKinds(listener, gateSupportedRouteKindsByProtocol)
		processedListener := Listener{
			K8sResource:       listener,
			AllowedRouteKinds: kinds,
			Conditions:        make(conditions.Conditions),
		}
		processedListeners = append(processedListeners, &processedListener)
	}
	treeGw.Listeners = processedListeners

	// Performs all needed checks
	// Only Checks and update listener status if the Gateway is Valid
	for _, listener := range treeGw.Listeners {
		switch listener.K8sResource.Protocol {
		// This switch will be completed with all needed checks per protocol
		case gatewayv1.HTTPProtocolType:
			listener.checkRouteGroupKind(treeGw, gateSupportedRouteKindsByProtocol)
		default:
			listener.checkProtocol(gateSupportedRouteKindsByProtocol)
		}
	}
}
