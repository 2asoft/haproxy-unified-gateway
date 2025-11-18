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
	"fmt"
	"log/slog"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/storage"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ Builder = &HTTPRouteBuilderImpl{}

type HTTPRouteBuilderImpl struct {
	mapsStorage storage.MapsStorage
	ControllerStore
	// runtimeUpdate indicates if the builder should trigger a runtime update of haproxy
	// when a map is changed
	runtimeUpdate bool
}

type HTTPRouteBuilderParams struct {
	storage.MapsStorage
	ControllerStore
	RuntimeUpdate bool
}

func NewHTTPRouteBuilder(params HTTPRouteBuilderParams) Builder {
	return &HTTPRouteBuilderImpl{
		ControllerStore: params.ControllerStore,
		mapsStorage:     params.MapsStorage,
		runtimeUpdate:   params.RuntimeUpdate,
	}
}

// --------------------
// GateTree Updates
// --------------------

func (b *HTTPRouteBuilderImpl) ComputeTreeUpdates() {
	b.addIndirectClusterStoreUpdates()
	b.computeGateTreeUpdates()
}

func (b *HTTPRouteBuilderImpl) addIndirectClusterStoreUpdates() {
	// Indirect from Services
	b.addIndirectMapsFromServices()
	// Indirect from Gateways
	b.addIndirectMapsFromGateways()
	// Indirect from Backend CR
	b.addIndirectMapsFromBackendCRs()
}

func (b *HTTPRouteBuilderImpl) addIndirectMapsFromServices() {
	for _, service := range b.ClusterStore.Updates.Services {
		b.addIndirectMapsFromService(service)
	}
}

func (b *HTTPRouteBuilderImpl) addIndirectMapsFromService(serviceUpdate store.Update[*v1.Service]) {
	addIndirectFromReferenced(
		serviceUpdate,
		b.ReferencedObjects.ReferencedServices,
		b.ClusterStore.HTTPRoutes,
		b.ClusterStore.Updates.HTTPRoutes,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeHTTPRoute),
		nil, // no ownerKey transformation
	)
}

func (b *HTTPRouteBuilderImpl) addIndirectMapsFromGateways() {
	for _, gateway := range b.ClusterStore.Updates.Gateways {
		b.addIndirectMapsFromGateway(gateway)
	}
}

func (b *HTTPRouteBuilderImpl) addIndirectMapsFromGateway(gatewayUpdate store.Update[*gatewayv1.Gateway]) {
	addIndirectFromReferenced(
		gatewayUpdate,
		b.ReferencedObjects.ReferencedGateways,
		b.ClusterStore.HTTPRoutes,
		b.ClusterStore.Updates.HTTPRoutes,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeHTTPRoute),
		nil,
	)
}

func (b *HTTPRouteBuilderImpl) addIndirectMapsFromBackendCRs() {
	for _, backendCR := range b.ClusterStore.Updates.BackendCRs {
		b.addIndirectMapsFromBackendCR(backendCR)
	}
}

func (b *HTTPRouteBuilderImpl) addIndirectMapsFromBackendCR(backendCRUpdate store.Update[*v3.Backend]) {
	addIndirectFromReferenced(
		backendCRUpdate,
		b.ReferencedObjects.ReferencedBackendCRs,
		b.ClusterStore.HTTPRoutes,
		b.ClusterStore.Updates.HTTPRoutes,
		b.ControllerStore.ExtractGVK(objtypes.ObjectTypeHTTPRoute),
		nil,
	)
}

func (b *HTTPRouteBuilderImpl) computeGateTreeUpdates() {
	for routeKey, routeUpdate := range b.ClusterStore.Updates.HTTPRoutes {
		b.computeTreeGatewayUpdate(routeKey, routeUpdate)
	}

	for _, httpRoute := range b.ControllerStore.GateTree.HTTPRoutes {
		if httpRoute == nil || httpRoute.TreeStatus.Status != store.StatusUpserted {
			continue
		}
		if httpRoute.isManaged() {
			// Process Rules
			b.buildRules(httpRoute)
		}

		// Merge the backendRef conditions
		httpRoute.mergeBackendConditions()
	}
}

func (b *HTTPRouteBuilderImpl) computeTreeGatewayUpdate(gwKey client.ObjectKey, routeUpdate store.Update[*gatewayv1.HTTPRoute]) {
	// here we check if its valid, if ref object exists and all checks
	var treeHTTPRoute *HTTPRoute
	alreadyManagedTreeRoute, alreadyManagedTreeRouteOK := b.GateTree.HTTPRoutes[gwKey]
	alreadyUnmanagedTreeRoute, alreadyUnmanagedTreeRouteOK := b.UnmanagedGateTree.HTTPRoutes[gwKey]

	if alreadyManagedTreeRouteOK {
		treeHTTPRoute = alreadyManagedTreeRoute
	} else if alreadyUnmanagedTreeRouteOK {
		treeHTTPRoute = alreadyUnmanagedTreeRoute
	}

	switch routeUpdate.Status {
	case store.StatusUpserted:
		if treeHTTPRoute != nil {
			treeHTTPRoute.SetAsUpserted(b.Logger, routeUpdate.NewObject)
			treeHTTPRoute.ResetChecks()
		} else {
			treeHTTPRoute = NewRoute(routeUpdate.NewObject, b.ControllerStore.ControllerName)
		}

		treeHTTPRoute.processChecks(b.ControllerStore)

		if treeHTTPRoute.isManaged() {
			b.SetAsManaged(treeHTTPRoute)
		} else {
			b.SetAsUnmanaged(treeHTTPRoute)
		}

		// Compute status only if managed HTTRoute
		// If not managed, then we should not update the status
		treeHTTPRoute.BuildConditions()

	case store.StatusDeleted:
		if treeHTTPRoute != nil {
			// Iterate over the listeners of the deleted route and remove the route from each listener
			for _, listeners := range treeHTTPRoute.Listeners.Iterate {
				for _, listener := range listeners {
					// This impact the Gateway object (listener status AttachedRoute), so it needs to be done, even so the HTTPRoute by itself is deleted
					listener.deleteAttachedRoute(client.ObjectKeyFromObject(treeHTTPRoute.K8sResource), b.ControllerStore)
				}
			}
			treeHTTPRoute.SetAsDeleted(b.Logger)
			treeHTTPRoute.ResetChecks()
		}
		// else nothing to do
		// It did not exists, it's deleted, noop
	}
}

func (r *HTTPRoute) isManaged() bool {
	return r.CheckParentRefs.Valid
}

func (r *HTTPRoute) ResetChecks() {
	r.CheckParentRefs = CheckResultRoute{}
	r.Listeners.Clear()
}

// -----------------------------------------------

func (b *HTTPRouteBuilderImpl) CleanTreeUpdates() {
	cleanTreeUpdates(b.GateTree.HTTPRoutes)
	cleanTreeUpdates(b.UnmanagedGateTree.HTTPRoutes)
}

func (b *HTTPRouteBuilderImpl) SetAsManaged(route *HTTPRoute) {
	b.Logger.LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprintf("%T MANAGED", *route),
		logging.LogAttrObjectKey(route.K8sResource))
	// Is it already in Managed
	key := client.ObjectKeyFromObject(route.K8sResource)
	b.ControllerStore.GateTree.HTTPRoutes[key] = route
	delete(b.ControllerStore.UnmanagedGateTree.Gateways, key)
}

func (b *HTTPRouteBuilderImpl) SetAsUnmanaged(route *HTTPRoute) {
	b.Logger.LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprintf("%T UNMANAGED", *route),
		logging.LogAttrObjectKey(route.K8sResource))
	// Is it already in Managed
	key := client.ObjectKeyFromObject(route.K8sResource)
	b.ControllerStore.UnmanagedGateTree.HTTPRoutes[key] = route
	delete(b.ControllerStore.GateTree.Gateways, key)
}

func (b *HTTPRouteBuilderImpl) buildRules(httpRoute *HTTPRoute) {
	httpRoute.Rules = make([]*HTTPRouteRule, 0)

	for _, rule := range httpRoute.K8sResource.Spec.Rules {
		treeRouteRule := HTTPRouteRule{
			K8sResource:     rule,
			CheckBackendRef: utils.NewKeyMap[gatewayv1.BackendObjectReference, CheckResult](utils.BackendObjectReferenceToKey),
		}
		httpRoute.Rules = append(httpRoute.Rules, &treeRouteRule)
	}

	// Performs all needed checks
	for _, rule := range httpRoute.Rules {
		rule.checkBackendRef(httpRoute, b.ControllerStore)
	}
}
