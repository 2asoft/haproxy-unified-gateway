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
	"strings"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	rc "github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/routes"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// A HTTPRoute represents a Kubernetes HTTPRoute
type HTTPRoute struct {
	// K8sResource is the source resource.
	K8sResource *gatewayv1.HTTPRoute
	// selected listener
	Listeners map[gatewayv1.ParentReference]*Listener // map[parentRef]
	// Final Conditions
	Conditions rc.RouteConditions
	// TreeStatus
	TreeStatus TreeUpdate[HTTPRoute]
	//
	ControllerName string
	// Management Checks
	CheckParamsRef CheckResultRoute
	// Valid
	BelongToOtherController bool
	Valid                   bool
}

// NewRoute creates a new Route for the GateTree.
func NewRoute(k8sObject *gatewayv1.HTTPRoute, controllerName string) *HTTPRoute {
	return &HTTPRoute{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[HTTPRoute]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
		Listeners: make(map[gatewayv1.ParentReference]*Listener),
		CheckParamsRef: CheckResultRoute{
			Valid: false,
			Conditions: rc.RouteConditions{
				Conditions:     make(map[gatewayv1.ParentReference]map[generic.ConditionType]generic.Condition),
				ControllerName: controllerName,
			},
		},
		ControllerName: controllerName,
	}
}

// SetAsUpserted marks the Secret as upserted in the GateTree.
func (r *HTTPRoute) SetAsUpserted(logger *slog.Logger, newK8sResource *gatewayv1.HTTPRoute) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Upserted",
		logging.LogAttrObjectKey(newK8sResource))
	r.TreeStatus.Status = store.StatusUpserted
	r.TreeStatus.OldTreeResource = r.DeepCopy()
	r.K8sResource = newK8sResource
}

// SetAsDeleted marks the Secret as deleted in the GateTree.
func (r *HTTPRoute) SetAsDeleted(logger *slog.Logger) {
	logger.LogAttrs(context.Background(), slog.LevelDebug, "TreeSecret Deleted",
		logging.LogAttrObjectKey(r.K8sResource))
	r.TreeStatus.Status = store.StatusDeleted
	r.TreeStatus.OldTreeResource = r.DeepCopy()
	r.K8sResource = nil
}

// DeepCopy creates a deep copy of the Secret.
func (r *HTTPRoute) DeepCopy() *HTTPRoute {
	if r == nil {
		return nil
	}
	// Save TreeStatus
	treeStatus := r.TreeStatus
	r.TreeStatus = TreeUpdate[HTTPRoute]{}

	// Manually handle the Listeners map
	listeners := r.Listeners
	r.Listeners = nil

	var copied HTTPRoute

	data, err := json.Marshal(r) //lint:ignore SA1026 We are manually handling the Listeners map to avoid this error.
	if err != nil {
		// Restore before returning
		r.Listeners = listeners
		r.TreeStatus = treeStatus
		return nil
	}
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct

	// Restore the original object
	r.Listeners = listeners
	r.TreeStatus = treeStatus

	// Manually copy the Listeners map
	if listeners != nil {
		copied.Listeners = make(map[gatewayv1.ParentReference]*Listener)
		for k, v := range listeners {
			copied.Listeners[k] = v.DeepCopy()
		}
	}

	return &copied
}

// processChecks processes the all checks for a HTTPRoute
func (r *HTTPRoute) processChecks(controllerStore ControllerStore) {
	r.checkParentRef(controllerStore)
	// g.checkGatewayClassIsValid(controllerStore)

	r.Valid = r.CheckParamsRef.Valid
}

//revive:disable:function-length,cognitive-complexity
func (r *HTTPRoute) checkParentRef(controllerStore ControllerStore) {
	parentrefs := r.K8sResource.Spec.ParentRefs
	// 1. if we do not have parent ref find the gateway with the hostname
	// 2. if we have parent ref find the gateway with the parent ref
	// 3. if we have multiple parent ref, do the same as 1 but among them
	// I might have multiple listeners, so maybe I needs to find all of them
	// Listener has also allowedRoutes CHECK
	if len(parentrefs) == 1 {
		parentRef := parentrefs[0]
		gwcKey := types.NamespacedName{
			Name: string(parentRef.Name),
		}
		if parentRef.Namespace != nil {
			gwcKey.Namespace = string(*parentRef.Namespace)
		} else {
			gwcKey.Namespace = r.K8sResource.Namespace
		}
		treeGw, ok := controllerStore.GateTree.Gateways[gwcKey]
		if ok && treeGw.TreeStatus.Status == store.StatusDeleted || !ok {
			// gateway exists, but it was deleted, so it is not valid
			r.CheckParamsRef.Valid = false
			r.CheckParamsRef.Conditions.Conditions[parentRef] = rc.ConditionNotAcceptedNoMatchingParent()
			delete(r.Listeners, parentRef)
			return
		}

		// listener name is called a section name and 'cool' part is that is optional
		// 1. we have it
		// 2. we don't have it
		found := false
		var selectedListener *Listener
		if parentRef.SectionName != nil {
			// treeGwc.Listeners contains kind
			for _, listener := range treeGw.Listeners {
				if listener.K8sResource.Name == *parentRef.SectionName {
					gvk := controllerStore.ExtractGVK(r.K8sResource)
					found := false
					for _, allowed := range listener.AllowedRouteKinds {
						if allowed.Group == (*gatewayv1.Group)(&gvk.Group) &&
							allowed.Kind == gatewayv1.Kind(gvk.Kind) {
							found = true
							break
						}
					}
					if !found {
						r.CheckParamsRef.Valid = false
						r.CheckParamsRef.Conditions.Conditions[parentRef] = rc.ConditionRouteReasonNotAllowedByListeners()
						delete(r.Listeners, parentRef)
						return
					}
					selectedListener = listener
					found = true
					break
				}
			}
		} else {
			// parentref.SectionName == nil
			gvk := controllerStore.ExtractGVK(r.K8sResource)
			for _, listener := range treeGw.Listeners {
				if !listener.Valid {
					continue
				}
				listenerHostname := listener.K8sResource.Hostname

				if matchHostname(r.K8sResource.Spec.Hostnames, listenerHostname) {
					found := false
					for _, allowed := range listener.AllowedRouteKinds {
						if allowed.Group == (*gatewayv1.Group)(&gvk.Group) &&
							allowed.Kind == gatewayv1.Kind(gvk.Kind) {
							found = true
							break
						}
					}
					if !found {
						// todo change reason because we have found it, but its not allowed
						continue
					}
					selectedListener = listener
					found = true
					break
				}
				// TODo, see if we need more that one listener
			}
		}

		if found {
			r.CheckParamsRef.Valid = true
			r.CheckParamsRef.Conditions.Conditions[parentRef] = rc.ConditionAccepted()
			r.Listeners[parentRef] = selectedListener
		} else {
			r.CheckParamsRef.Valid = false
			r.CheckParamsRef.Conditions.Conditions[parentRef] = rc.ConditionNotAcceptedNoMatchingHostname()
			delete(r.Listeners, parentRef)
		}
		return
	}
	var gateways []*Gateway
	if len(parentrefs) > 0 {
		for _, parentRef := range parentrefs {
			gwKey := types.NamespacedName{
				Name: string(parentRef.Name),
			}
			if parentRef.Namespace != nil {
				gwKey.Namespace = string(*parentRef.Namespace)
			}
			treeGw, ok := controllerStore.GateTree.Gateways[gwKey]
			if ok && treeGw.Valid {
				gateways = append(gateways, treeGw)
			}
		}
	} else {
		for _, treeGw := range controllerStore.GateTree.Gateways {
			if treeGw.Valid {
				gateways = append(gateways, treeGw)
			}
		}
	}
	// now that we have all the gateways, check if any route hostname matches any listener hostname
	for _, gw := range gateways {
		var matched bool
		for _, listener := range gw.Listeners {
			if !listener.Valid {
				continue
			}
			if matchHostname(r.K8sResource.Spec.Hostnames, listener.K8sResource.Hostname) {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
	}
	// what happens if we have a match for more of them ?
	// if len(matchedGateways) == 0 {
	//   controllerStore.GateTree.HTTPRoutes[r.K8sResourceKey] = r
	// }
}

// matchHostname checks if a route's hostnames match a listener's hostname.
// The rules are based on the Gateway API specification.
func matchHostname(routeHostnames []gatewayv1.Hostname, listenerHostname *gatewayv1.Hostname) bool {
	// If the listener hostname is not set, it matches any route hostname.
	if listenerHostname == nil || *listenerHostname == "" {
		return true
	}

	for _, routeHostname := range routeHostnames {
		if routeHostname == "" {
			return true
		}
		if match(string(routeHostname), string(*listenerHostname)) {
			return true
		}
	}
	return false
}

// match performs the actual hostname matching between a route and a listener hostname.
func match(routeHostname, listenerHostname string) bool {
	// Exact match
	if routeHostname == listenerHostname {
		return true
	}

	// Wildcard match for listener
	if strings.HasPrefix(listenerHostname, "*.") {
		domain := strings.TrimPrefix(listenerHostname, "*.")
		if routeHostname != domain && strings.HasSuffix(routeHostname, "."+domain) {
			return true
		}
	}

	// Wildcard match for route
	if strings.HasPrefix(routeHostname, "*.") {
		domain := strings.TrimPrefix(routeHostname, "*.")
		if listenerHostname != domain && strings.HasSuffix(listenerHostname, "."+domain) {
			return true
		}
	}

	return false
}

func (r *HTTPRoute) BuildConditions() {
	if !r.isManaged() {
		return
	}
	r.Conditions = rc.RouteConditions{}
	r.Conditions.Conditions = make(map[gatewayv1.ParentReference]map[generic.ConditionType]generic.Condition)
	// If not valid, we do not need to continue
	if !r.CheckParamsRef.Valid {
		r.Valid = false
		r.Conditions.SetGeneration(r.K8sResource.Generation)
		r.Conditions.MergeOverrideConditions(r.CheckParamsRef.Conditions)
		return
	}

	r.Conditions.ControllerName = r.ControllerName
	r.Conditions.SetGeneration(r.K8sResource.Generation)
	for _, parentRef := range r.K8sResource.Spec.ParentRefs {
		r.Conditions.Conditions[parentRef] = make(map[generic.ConditionType]generic.Condition)
		_, exists := r.Listeners[parentRef]
		if exists {
			r.Conditions.Conditions[parentRef] = rc.ConditionAccepted()
			continue
		}
		r.Conditions.Conditions[parentRef] = rc.ConditionNotAcceptedNoMatchingParent()
	}
}
