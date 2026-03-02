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
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/protocols"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type listenerRefs struct {
	gatewayRef  *gatewayv1.Gateway
	listenerRef gatewayv1.Listener
}

// computeListenerConflicts detects conflicts between listeners on the same port.
// It populates b.mapPort2ListenerConflict with the results of the conflict detection.
// Conflicts can arise from:
// - Different protocol categories (e.g., HTTP vs HTTPS) on the same port.
// - Overlapping hostnames for listeners with compatible protocols.
// The conflict resolution strategy favors the oldest Gateway (by CreationTimestamp).
func (b *GatewayBuilderImpl) computeListenerConflicts() {
	// 0- Compute conflicts only between non-deleted Gateways and their listeners. Deleted Gateways and their listeners are ignored in the conflict detection.
	// 1- Sort all non-deleted Gateways by creation timestamp
	sortedGws := utils.MapToSortedListByCreationTimestamp(b.nonDeletedGateways())

	portListeners := b.listenersPerPort(sortedGws)

	// In portListeners we have now all listeners grouped by port
	// Some of them are already marked as conflicting because of protocol category difference
	// Now we need to check for hostname overlaps
	for _, listenerRefs := range portListeners {
		// The first listener in the list is the winner due to the gateway creation timestamp sort.
		// All subsequent listeners are checked against this winner.
		if len(listenerRefs) > 1 {
			winner := listenerRefs[0]
			winnerHostname := utils.PointerDefaultValueIfNil(winner.listenerRef.Hostname)
			winnerLk := NewListenerKey(winner.gatewayRef, winner.listenerRef)

			if _, ok := b.ControllerStore.mapPort2Listeners[winner.listenerRef.Port]; !ok {
				b.ControllerStore.mapPort2Listeners[winner.listenerRef.Port] = make(map[client.ObjectKey]listenerConflictCondition)
			}
			b.ControllerStore.mapPort2Listeners[winner.listenerRef.Port][winnerLk] = listenerConflictCondition{
				hasConflict: false,
				reason:      "",
				protocol:    protocols.ProtocolCategories[winner.listenerRef.Protocol],
			}

			for i := 1; i < len(listenerRefs); i++ {
				challenger := listenerRefs[i]
				challengerHostname := utils.PointerDefaultValueIfNil(challenger.listenerRef.Hostname)
				challengerLk := NewListenerKey(challenger.gatewayRef, challenger.listenerRef)

				lcc := listenerConflictCondition{
					hasConflict: false,
					reason:      "",
					protocol:    protocols.ProtocolCategories[challenger.listenerRef.Protocol],
				}
				if overlaps(string(winnerHostname), string(challengerHostname)) {
					lcc = listenerConflictCondition{
						hasConflict: true,
						reason:      string(gatewayv1.ListenerReasonHostnameConflict),
						protocol:    protocols.ProtocolCategories[challenger.listenerRef.Protocol],
					}
				}
				b.ControllerStore.mapPort2Listeners[winner.listenerRef.Port][challengerLk] = lcc
			}
		} else {
			lk := NewListenerKey(listenerRefs[0].gatewayRef, listenerRefs[0].listenerRef)
			// Only one listener on this port, no conflict
			if _, ok := b.ControllerStore.mapPort2Listeners[listenerRefs[0].listenerRef.Port]; !ok {
				b.ControllerStore.mapPort2Listeners[listenerRefs[0].listenerRef.Port] = make(map[client.ObjectKey]listenerConflictCondition)
			}
			b.ControllerStore.mapPort2Listeners[listenerRefs[0].listenerRef.Port][lk] = listenerConflictCondition{
				hasConflict: false,
				reason:      "",
				protocol:    protocols.ProtocolCategories[listenerRefs[0].listenerRef.Protocol],
			}
		}
	}
}

// nonDeletedGateways returns a filtered copy of b.GateTree.Gateways containing
// only Gateways that have not been marked for deletion.
func (b *GatewayBuilderImpl) nonDeletedGateways() map[types.NamespacedName]*Gateway {
	result := make(map[types.NamespacedName]*Gateway, len(b.GateTree.Gateways))
	for k, gw := range b.GateTree.Gateways {
		if gw == nil || gw.K8sResource == nil {
			continue
		}
		if gw.TreeStatus.Status == store.StatusDeleted {
			continue
		}
		result[k] = gw
	}
	return result
}

// listenersPerPort groups listeners by port.
// It ensures that all listeners on the same port share a compatible protocol category (e.g., all HTTP or all HTTPS).
// Listeners with incompatible protocols are marked as conflicting in mapPort2ListenerConflict
// and are not included in the returned map.
func (b *GatewayBuilderImpl) listenersPerPort(sortedGws []*Gateway) map[gatewayv1.PortNumber][]listenerRefs {
	portListeners := make(map[gatewayv1.PortNumber][]listenerRefs)
	mapPort2ProtocolCategory := make(map[gatewayv1.PortNumber]protocols.ProtocolCategory)

	// 2- For each Gateway, for each listener, check if the port is conflicting with some other port
	for _, treeGw := range sortedGws {
		if treeGw == nil || treeGw.K8sResource == nil {
			// ... deleted
			continue
		}
		if treeGw.TreeStatus.Status == store.StatusDeleted {
			continue
		}

		for _, listener := range treeGw.K8sResource.Spec.Listeners {
			// Compute the gatewaylistener key (reference to both Gateway and Listener)
			lk := NewListenerKey(treeGw.K8sResource, listener)

			// Check if the protocol category (secure/insecure/...) for the port is different from the current listener's protocol.
			_, ok := mapPort2ProtocolCategory[listener.Port]
			if !ok {
				// First listener on this port, the port protocol category is set for this port (secure/insecure/...)
				mapPort2ProtocolCategory[listener.Port] = protocols.ProtocolCategories[listener.Protocol]
				portListeners[listener.Port] = append(portListeners[listener.Port], listenerRefs{
					gatewayRef:  treeGw.K8sResource,
					listenerRef: listener,
				})
				continue
			}
			// Second (or more) listener on this port
			if mapPort2ProtocolCategory[listener.Port] != protocols.ProtocolCategories[listener.Protocol] {
				// If the protocol category is different, it's a conflict
				// This listener is conflicting
				if _, ok := b.ControllerStore.mapPort2Listeners[listener.Port]; !ok {
					b.ControllerStore.mapPort2Listeners[listener.Port] = make(map[client.ObjectKey]listenerConflictCondition)
				}
				lcc := listenerConflictCondition{
					hasConflict: true,
					reason:      string(gatewayv1.ListenerReasonProtocolConflict),
					protocol:    protocols.ProtocolCategories[listener.Protocol],
				}
				b.ControllerStore.mapPort2Listeners[listener.Port][lk] = lcc
				continue
			}
			// Same protocol category, check for overlap
			portListeners[listener.Port] = append(portListeners[listener.Port], listenerRefs{
				gatewayRef:  treeGw.K8sResource,
				listenerRef: listener,
			})
		}
	}
	return portListeners
}
