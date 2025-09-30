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
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	rc "github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/routes"
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// A HTTPRoute represents a Kubernetes HTTPRoute
type HTTPRoute struct {
	// K8sResource is the source resource.
	K8sResource *gatewayv1.HTTPRoute
	// selected listener
	Listeners utils.KeyMap[gatewayv1.ParentReference, *Listener] // map[parentRef]
	// Final Conditions
	Conditions rc.RouteConditions
	// TreeStatus
	TreeStatus     TreeUpdate[HTTPRoute]
	ControllerName string
	// Management Checks
	// CheckParentRefs will only contains conditions for parents (Gateways) managed by us
	CheckParentRefs CheckResultRoute
	// Valid
	Valid bool
}

// NewRoute creates a new Route for the GateTree.
func NewRoute(k8sObject *gatewayv1.HTTPRoute, controllerName string) *HTTPRoute {
	return &HTTPRoute{
		K8sResource: k8sObject,
		TreeStatus: TreeUpdate[HTTPRoute]{
			Status:          store.StatusUpserted,
			OldTreeResource: nil,
		},
		Listeners: utils.NewKeyMap[gatewayv1.ParentReference, *Listener](utils.ParentRefToKey),
		CheckParentRefs: CheckResultRoute{
			Valid: false,
			Conditions: rc.RouteConditions{
				Conditions:     utils.NewKeyMap[gatewayv1.ParentReference, generic.Conditions](utils.ParentRefToKey),
				ControllerName: controllerName,
			},
		},
		ControllerName: controllerName,
	}
}

// SetAsUpserted marks the HTTPRoute as upserted in the GateTree.
func (r *HTTPRoute) SetAsUpserted(logger *slog.Logger, newK8sResource *gatewayv1.HTTPRoute) {
	setResourceStatus(logger, r, newK8sResource, store.StatusUpserted)
	r.K8sResource = newK8sResource
}

// SetAsDeleted marks the Secret as deleted in the GateTree.
func (r *HTTPRoute) SetAsDeleted(logger *slog.Logger) {
	setResourceStatus(logger, r, nil, store.StatusDeleted)
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
	r.Listeners.Clear()

	var copied HTTPRoute

	data, err := json.Marshal(r)
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
	copied.Listeners = listeners.DeepCopy()

	return &copied
}

// GetTreeStatus returns the TreeStatus of the HTTPRoute.
func (r *HTTPRoute) GetTreeStatus() *TreeUpdate[HTTPRoute] {
	return &r.TreeStatus
}

// SetTreeStatus sets the TreeStatus of the HTTPRoute.
func (r *HTTPRoute) SetTreeStatus(treeStatus TreeUpdate[HTTPRoute]) {
	r.TreeStatus = treeStatus
}

// processChecks processes the all checks for a HTTPRoute
func (r *HTTPRoute) processChecks(controllerStore ControllerStore) {
	r.checkParentRefs(controllerStore)
	// g.checkGatewayClassIsValid(controllerStore)

	r.Valid = r.CheckParentRefs.Valid
}

//revive:disable:function-length,cognitive-complexity
func (r *HTTPRoute) checkParentRefs(controllerStore ControllerStore) {
	routeConditions := rc.RouteConditions{
		ControllerName: r.ControllerName,
		Conditions:     utils.NewKeyMap[gatewayv1.ParentReference, generic.Conditions](utils.ParentRefToKey),
	}
	checkParentsRefs := CheckResultRoute{
		Valid:      false,
		Conditions: routeConditions,
	}

	if r.K8sResource == nil {
		r.CheckParentRefs = checkParentsRefs
		return
	}
	parentrefs := r.K8sResource.Spec.ParentRefs
	atLeastOneValidParentRef := false
	for _, parentRef := range parentrefs {
		// 1. if we do not have parent ref find the gateway with the hostname
		// --- This loop will do nothing and we will have no parentRef condition, the route is not attached
		// 2. if we have parent ref find the gateway listener with the parent ref
		//    Note that the parentRef.Name (the Gateway name) is mandatory, only sectionName is optional
		// I might have multiple listeners, so maybe I needs to find all of them
		// Listener has also allowedRoutes CHECK
		result := r.checkParentRef(parentRef, controllerStore)

		if !result.Managed {
			// do not add the parentRef in Listener not check result
			continue
		}

		// From now on, parentRef is a managed Gateway

		// 1- The parentRef is not Valid
		if !result.Valid {
			// do not add the parentREf in Listener or check result
			routeConditions.MergeOverrideConditionsForParentRef(parentRef, result.Conditions)
			continue
		}

		// 2- The parentRef is Valid
		atLeastOneValidParentRef = true
		routeConditions.MergeOverrideConditionsForParentRef(parentRef, result.Conditions)
		// no specific conditions
		r.Listeners.Set(parentRef, result.Listener)
	}

	// Gather all Conditions for all parentRefs in the result
	checkParentsRefs.Conditions = routeConditions

	// If at least one parentRef is valid, the whole check is Valid
	if atLeastOneValidParentRef {
		checkParentsRefs.Valid = true
	}

	r.CheckParentRefs = checkParentsRefs
}

type checkParentRefResult struct {
	// Conditions has a set of failing conditions
	// It is set only if Managed is true
	Conditions generic.Conditions
	// Listener is the selected Listener
	Listener *Listener
	// Managed is true if the Gateway is managed by our controller
	Managed bool
	// Valid is set only if Managed is true
	Valid bool
}

// checkParentRef checks 1 parentRef and returns:
// - a bool indicating if the parentRef is valid, and if it's not, a set of conditions detailing why
func (r *HTTPRoute) checkParentRef(parentRef gatewayv1.ParentReference, controllerStore ControllerStore) checkParentRefResult {
	if !isParentRefGroupKindSupported(parentRef, controllerStore.ExtractGVK) {
		return checkParentRefResult{
			Managed:    false,
			Valid:      false,
			Conditions: generic.Conditions{},
		}
	}

	gwKey := GetParentRefNamespacedName(parentRef, r.K8sResource)

	treeGw, ok := controllerStore.GateTree.Gateways[gwKey]
	if ok && treeGw.TreeStatus.Status == store.StatusDeleted {
		// gateway exists, but it was deleted
		return checkParentRefResult{
			Managed:    true,
			Valid:      false,
			Conditions: rc.ConditionNotAcceptedNoMatchingParent(),
		}
	}
	if !ok {
		// It's a whole different story, it means the Gateway does not exists,
		// we can not know if it's managed by our controller or not
		return checkParentRefResult{
			Managed:    false,
			Valid:      false,
			Conditions: generic.Conditions{},
		}
	}

	// From now on, the parent references an exising Gateway managed by our controller
	// SectionName is optional
	// 1- if set, the only attachable listener is the one references by the sectionName
	// 2- if not set, all listeners from the Gateway are attachable
	attachableListeners := make([]*Listener, 0)
	if parentRef.SectionName != nil {
		// Check if the Gateway has this listener
		listener, ok := treeGw.Listeners[string(*parentRef.SectionName)]
		if !ok {
			return checkParentRefResult{
				Managed:    true,
				Valid:      false,
				Conditions: rc.ConditionNotAcceptedNoMatchingParent(),
			}
		}
		// We found the listener, it does exists
		attachableListeners = append(attachableListeners, listener)
	} else {
		for _, listener := range treeGw.Listeners {
			attachableListeners = append(attachableListeners, listener)
		}
	}

	// If we have only 1 attache listener, just check the hostnames
	if len(attachableListeners) == 1 {
		listener := attachableListeners[0]
		if !listener.Valid {
			return checkParentRefResult{
				Managed:    true,
				Valid:      false,
				Conditions: rc.ConditionNotAcceptedNoMatchingParent(),
			}
		}
		if matched := matchHostname(r.K8sResource.Spec.Hostnames, listener.K8sResource.Hostname); matched {
			// AllowedRouteKind ???
			allowedRouteKind := r.isAllowedRouteKind(listener, controllerStore.ExtractGVK)
			if !allowedRouteKind {
				return checkParentRefResult{
					Managed:    true,
					Valid:      false,
					Conditions: rc.ConditionNotAcceptedRouteReasonNotAllowedByListeners(),
				}
			}

			// Set the Listener attached Route
			listener.addAttachedRoute(client.ObjectKeyFromObject(r.K8sResource), controllerStore)

			return checkParentRefResult{
				Managed:    true,
				Valid:      true,
				Listener:   listener,
				Conditions: rc.ConditionAccepted(),
			}
		}
		return checkParentRefResult{
			Managed:    true,
			Valid:      false,
			Conditions: rc.ConditionNotAcceptedNoMatchingHostname(),
		}
	}

	// Now, case where we need to find a listener among the list
	var matched bool
	var matchedListener *Listener
	for _, listener := range attachableListeners {
		if !listener.Valid {
			continue
		}

		// Match Hostname ?
		if matchHostname(r.K8sResource.Spec.Hostnames, listener.K8sResource.Hostname) {
			matched = true
			matchedListener = listener
			break
		}
	}
	if !matched {
		return checkParentRefResult{
			Managed:    true,
			Valid:      false,
			Conditions: rc.ConditionNotAcceptedNoMatchingParent(),
		}
	}

	// We have found a listener
	// Allowed RouteKind ??
	allowedRouteKind := r.isAllowedRouteKind(matchedListener, controllerStore.ExtractGVK)
	// Set the Listener attached Route
	matchedListener.addAttachedRoute(client.ObjectKeyFromObject(r.K8sResource), controllerStore)

	if !allowedRouteKind {
		return checkParentRefResult{
			Managed:    true,
			Valid:      false,
			Conditions: rc.ConditionNotAcceptedRouteReasonNotAllowedByListeners(),
		}
	}

	return checkParentRefResult{
		Managed:    true,
		Valid:      true,
		Listener:   matchedListener,
		Conditions: generic.Conditions{},
	}
}

func (r *HTTPRoute) isAllowedRouteKind(listener *Listener, extractGVK utils.ExtractGVK) bool {
	gvk := extractGVK(r.K8sResource)
	for _, allowed := range listener.AllowedRouteKinds {
		if allowed.Group != nil && *allowed.Group == gatewayv1.Group(gvk.Group) {
			if allowed.Kind == gatewayv1.Kind(gvk.Kind) {
				return true
			}
		}
		if allowed.Group == nil && allowed.Kind == gatewayv1.Kind(gvk.Kind) {
			return true
		}
	}
	return false
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
	r.Conditions = rc.RouteConditions{
		ControllerName: r.ControllerName,
		Conditions:     utils.NewKeyMap[gatewayv1.ParentReference, generic.Conditions](utils.ParentRefToKey),
	}
	r.Conditions.SetGeneration(r.K8sResource.Generation)

	r.CheckParentRefs.Conditions.Conditions.Iterate(func(key string, parentRefConds generic.Conditions) bool {
		parentRef, err := utils.KeyToParentRef(key)
		if err != nil {
			return true // continue iteration
		}
		r.Conditions.MergeOverrideConditionsForParentRef(parentRef, parentRefConds)
		return true
	})
	// This needs to change when we implement more checks
	r.Valid = r.CheckParentRefs.Valid
}

// GetParentRefNamespacedName returns the namespaced name for a parentRef reference,
// using the Route's namespace as a default if the reference does not specify one.
func GetParentRefNamespacedName(parentRef gatewayv1.ParentReference, route *gatewayv1.HTTPRoute) types.NamespacedName {
	return types.NamespacedName{
		Namespace: getNamespace(parentRef.Namespace, route),
		Name:      string(parentRef.Name),
	}
}

// GetBackendRefNamespacedName returns the namespaced name for a backendref reference,
// using the Route's namespace as a default if the reference does not specify one.
func GetBackendRefNamespacedName(backendRef gatewayv1.BackendObjectReference, route *gatewayv1.HTTPRoute) types.NamespacedName {
	return types.NamespacedName{
		Namespace: getNamespace(backendRef.Namespace, route),
		Name:      string(backendRef.Name),
	}
}

// getNamespace returns
//   - the route's namespace if ns if nil
//   - *ns if not nil
func getNamespace(ns *gatewayv1.Namespace, route *gatewayv1.HTTPRoute) string {
	if ns != nil {
		return string(*ns)
	}
	return route.Namespace
}

// isParentRefGroupKindSupported checks if the provided HTTPRoute parent reference has a supported Group and Kind.
// It only supports `gatewayv1.Gateway` resources.
func isParentRefGroupKindSupported(parentRef gatewayv1.ParentReference, extractGVK utils.ExtractGVK) bool {
	gatewaytype := objtypes.ObjectTypeGateway
	gatewayGVK := extractGVK(gatewaytype)
	if parentRef.Kind != nil && *parentRef.Kind != gatewayv1.Kind(gatewayGVK.Kind) {
		return false
	}
	if parentRef.Group != nil && *parentRef.Group != gatewayv1.Group(gatewayGVK.Group) {
		return false
	}
	return true
}

// isBackendRefGroupKindSupported checks if the provided HTTPRoute parent reference has a supported Group and Kind.
// It only supports `corev1.Service` resources.
func isBackendRefGroupKindSupported(backendRef gatewayv1.BackendObjectReference, extractGVK utils.ExtractGVK) bool {
	servicetype := objtypes.ObjectTypeService
	serviceGVK := extractGVK(servicetype)
	if backendRef.Kind != nil && *backendRef.Kind != gatewayv1.Kind(serviceGVK.Kind) {
		return false
	}
	if backendRef.Group != nil && *backendRef.Group != gatewayv1.Group(serviceGVK.Group) {
		return false
	}
	return true
}
