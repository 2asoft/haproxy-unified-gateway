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
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/conditions"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/conditions/generic"
	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Listener struct {
	// K8sResource is the source resource.
	K8sResource gatewayv1.Listener
	// Final Conditions
	Conditions     generic.Conditions
	AttachedRoutes AttachedRoutes
	// Owner is the gateway that this listener is connected to
	Owner client.ObjectKey
	// Checks results
	CheckRouteGroupKind CheckResult
	CheckProtocol       CheckResult
	CheckSecret         CheckResult
	CheckConflict       CheckResult
	// AllowedRouteKinds is the list of allowed route kinds for this listener.
	AllowedRouteKinds []gatewayv1.RouteGroupKind
	// Valid
	Valid bool
}

// DeepCopy creates a deep copy of the Listener.
func (l *Listener) DeepCopy() *Listener {
	if l == nil {
		return nil
	}

	var copied Listener
	// We can ignore the error here, as we are controlling the input
	data, _ := json.Marshal(l)
	_ = json.Unmarshal(data, &copied)

	return &copied
}

type RouteGroupKind struct {
	// Group is the group of the RouteGroupKind.
	Group gatewayv1.Group `json:"group,omitempty"`
	// Kind is the kind of the RouteGroupKind.
	Kind gatewayv1.Kind `json:"kind"`
}

var gateSupportedRouteKindsByProtocol = map[gatewayv1.ProtocolType][]gatewayv1.RouteGroupKind{
	gatewayv1.HTTPProtocolType:  {objtypes.RouteKindHTTP},
	gatewayv1.HTTPSProtocolType: {objtypes.RouteKindHTTP},
	gatewayv1.TLSProtocolType:   {objtypes.RouteKindTLS},
}

func supportedKinds(listener gatewayv1.Listener, gateSupportedRouteKinds map[gatewayv1.ProtocolType][]gatewayv1.RouteGroupKind) []gatewayv1.RouteGroupKind {
	gateSupportedRouteKindsForProtocol := gateSupportedRouteKinds[listener.Protocol]
	kinds := []gatewayv1.RouteGroupKind{}

	// If the listener does not have allowedRoutes.Kinds defined, then list of supported routes types only depends
	// on the listener protocol and is what is supported by the gateway.
	if listener.AllowedRoutes == nil || len(listener.AllowedRoutes.Kinds) == 0 {
		kinds = append(kinds, gateSupportedRouteKindsForProtocol...)

		slices.SortFunc(kinds, sortRouteKinds)
		return kinds
	}

	// If the listener has some allowedRoutes.Kinds, we take the intersection
	// of what the listener allows and what the gateway supports for the protocol.
	if listener.AllowedRoutes != nil {
		for _, listenerAllowedKind := range listener.AllowedRoutes.Kinds {
			if ok := isSupportedProtocolRouteKind(listenerAllowedKind, gateSupportedRouteKindsForProtocol); ok {
				kinds = append(kinds, listenerAllowedKind)
			}
		}
	}

	slices.SortFunc(kinds, sortRouteKinds)
	return kinds
}

func sortRouteKinds(a, b gatewayv1.RouteGroupKind) int {
	// A nil group defaults to gateway.networking.k8s.io
	groupA := gatewayv1.GroupName
	if a.Group != nil {
		groupA = string(*a.Group)
	}
	groupB := gatewayv1.GroupName
	if b.Group != nil {
		groupB = string(*b.Group)
	}
	// Compare by Group first, then by Kind.
	if c := cmp.Compare(groupA, groupB); c != 0 {
		return c
	}
	return cmp.Compare(string(a.Kind), string(b.Kind))
}

func isSupportedProtocolRouteKind(kind gatewayv1.RouteGroupKind, supportedRouteKinds []gatewayv1.RouteGroupKind) bool {
	if kind.Group != nil && *kind.Group != gatewayv1.GroupName {
		return false
	}
	for _, k := range supportedRouteKinds {
		if k.Kind == kind.Kind {
			return true
		}
	}

	return false
}

func (l *Listener) checkRouteGroupKind(treeGw *Gateway, gateSupportedRouteKinds map[gatewayv1.ProtocolType][]gatewayv1.RouteGroupKind) {
	if !treeGw.Valid {
		l.CheckRouteGroupKind = CheckResult{}
		return
	}

	listener := l.K8sResource
	gateSupportedRouteKindsForProtocol := gateSupportedRouteKinds[listener.Protocol]

	// If the listener does not have allowedRoutes.Kinds defined, then list of supported routes types only depends
	// on the listener protocol and is what is supported by the gateway.
	if listener.AllowedRoutes == nil || len(listener.AllowedRoutes.Kinds) == 0 {
		l.CheckRouteGroupKind = CheckResult{
			Valid: true,
		}
		return
	}

	// If the listener has some allowedRoutes.Kinds, we take the intersection
	// of what the listener allows and what the gateway supports for the protocol.
	unsupportedRoute := ""
	if listener.AllowedRoutes != nil {
		for _, listenerAllowedKind := range listener.AllowedRoutes.Kinds {
			if ok := isSupportedProtocolRouteKind(listenerAllowedKind, gateSupportedRouteKindsForProtocol); !ok {
				unsupportedRoute = string(listenerAllowedKind.Kind)
			}
		}
	}

	checkResult := CheckResult{}
	if unsupportedRoute != "" {
		acceptedKinds := make([]gatewayv1.RouteGroupKind, 0, len(gateSupportedRouteKindsForProtocol))
		acceptedKinds = append(acceptedKinds, gateSupportedRouteKindsForProtocol...)
		slices.SortFunc(acceptedKinds, sortRouteKinds)

		msg := fmt.Sprintf("%s is not supported. Supported Route Kinds for this protocol %s", unsupportedRoute, utils.RouteGroupKindsToString(acceptedKinds))
		checkResult.Conditions = conditions.NewListenerResolvedRefInvalidRouteKinds(msg)
		checkResult.Valid = false

		l.CheckRouteGroupKind = checkResult
		return
	}

	l.CheckRouteGroupKind = CheckResult{
		Valid: true,
	}
}

func (l *Listener) checkProtocol(gateSupportedRouteKinds map[gatewayv1.ProtocolType][]gatewayv1.RouteGroupKind) {
	listener := l.K8sResource
	gateSupportedRouteKindsForProtocol := gateSupportedRouteKinds[listener.Protocol]
	// UnsupportedProtocol
	if len(gateSupportedRouteKindsForProtocol) == 0 {
		supportedProtocols := make([]string, 0, len(gateSupportedRouteKinds))

		for protocol := range gateSupportedRouteKinds {
			supportedProtocols = append(supportedProtocols, string(protocol))
		}
		slices.Sort(supportedProtocols)

		valErr := field.NotSupported(
			field.NewPath("protocol"),
			listener.Protocol,
			supportedProtocols,
		)
		l.CheckProtocol = CheckResult{
			Valid:      false,
			Conditions: conditions.NewListenerAcceptedUnsupportedProtocol(valErr.Error()),
		}
		return
	}

	l.CheckProtocol = CheckResult{
		Valid: true,
	}
}

func (l *Listener) checkCertificateRefs(treeGw *Gateway, gateSecrets map[types.NamespacedName]*Secret) {
	if !treeGw.Valid {
		l.CheckSecret = CheckResult{}
		return
	}

	listener := l.K8sResource
	if listener.TLS == nil || len(listener.TLS.CertificateRefs) == 0 {
		l.CheckSecret = CheckResult{
			Valid: true,
		}
		return
	}

	// Check is the secret exists
	// We only accept Secret as CertificateRefs
	// We only accept v1.Secret
	for _, certRef := range listener.TLS.CertificateRefs {
		if !l.isSupportedCertKindGroup(certRef) {
			msg := "Listener CertificateRefs must be of Group/Kind Secret"
			l.CheckSecret = CheckResult{
				Valid:      false,
				Conditions: conditions.NewListenerResolvedRefInvalidCertificateRefs(msg),
			}
			break
		}

		nsName := GetCertificateRefNamespacedName(certRef, treeGw.K8sResource)
		treeSecret, ok := gateSecrets[nsName]
		if !ok || treeSecret.TreeStatus.Status == store.StatusDeleted {
			msg := fmt.Sprintf("Secret %s/%s does not exist", nsName.Namespace, nsName.Name)
			l.CheckSecret = CheckResult{
				Valid:      false,
				Conditions: conditions.NewListenerResolvedRefInvalidCertificateRefs(msg),
			}
		}
	}
}

func (l *Listener) checkConflict(treeGw *Gateway, multipleListenersPerPort map[gatewayv1.PortNumber][]gatewaylistener) {
	if !treeGw.Valid {
		l.CheckConflict = CheckResult{}
		return
	}
	listener := l.K8sResource
	if gls, ok := multipleListenersPerPort[listener.Port]; ok {
		// There are multiple listeners on this port
		// Should be rejected. It's not allowed to pick one of the listeners
		// All listeners should be rejects.
		// The Gateway by itself should be accepted only if there is at least 1 valid Listener remaining
		// after rejecting all invalid listeners
		conflictingKeys := make([]string, 0, len(gls))
		for _, gl := range gls {
			conflictingKeys = append(conflictingKeys, gl.listenerKey.String())
		}
		slices.Sort(conflictingKeys)
		msg := fmt.Sprintf("Conflicting listeners: %s", strings.Join(conflictingKeys, ", "))
		cond := conditions.NewListenerConflicted(msg)
		l.CheckConflict = CheckResult{
			Valid:      false,
			Conditions: cond,
		}
	}
}

func (*Listener) isSupportedCertKindGroup(certRef gatewayv1.SecretObjectReference) bool {
	supportedKind := certRef.Kind == nil || *certRef.Kind == "Secret"
	supportedGroup := certRef.Group == nil || *certRef.Group == ""
	return supportedKind && supportedGroup
}

func (l *Listener) BuildConditions(treeGw *Gateway) {
	l.Conditions = make(generic.Conditions)
	l.Conditions.MergeOverrideConditions(l.CheckRouteGroupKind.Conditions)
	l.Conditions.MergeOverrideConditions(l.CheckProtocol.Conditions)
	l.Conditions.MergeOverrideConditions(l.CheckSecret.Conditions)
	l.Conditions.MergeOverrideConditions(l.CheckConflict.Conditions)
	// Should we process with Haproxy programmation
	shouldProgramm := true

	_, exists := l.Conditions.GetCondition(generic.ConditionType(gatewayv1.ListenerConditionAccepted))
	if !exists {
		// Accepted = OK
		l.Conditions.MergeOverrideConditions(conditions.NewListenerAcceptedOK())
	} else {
		l.Conditions.MergeOverrideConditions(conditions.NewListenerProgrammedInvalid())
		shouldProgramm = false
	}

	_, exists = l.Conditions.GetCondition(generic.ConditionType(gatewayv1.ListenerConditionResolvedRefs))
	if !exists {
		// ResolvedRefs = OK
		l.Conditions.MergeOverrideConditions(conditions.NewListenerResolvedRefOK())
	} else {
		l.Conditions.MergeOverrideConditions(conditions.NewListenerProgrammedInvalid())
		shouldProgramm = false
	}

	_, exists = l.Conditions.GetCondition(generic.ConditionType(gatewayv1.ListenerConditionConflicted))
	if exists {
		l.Conditions.MergeOverrideConditions(conditions.NewListenerProgrammedInvalid())
		shouldProgramm = false
	}

	if shouldProgramm {
		l.Conditions.MergeOverrideConditions(conditions.NewListenerProgrammedPending())
	}

	l.Valid = shouldProgramm
	l.Conditions.SetGeneration(treeGw.K8sResource.GetGeneration())
}

func (l *Listener) resetChecks() {
	l.CheckRouteGroupKind = CheckResult{}
	l.CheckProtocol = CheckResult{}
	l.CheckSecret = CheckResult{}
	l.CheckConflict = CheckResult{}
}

// ListenerKey returns the Listener owner key appending the listener name to it
// For Gateway ns/gateway, if the Listener name is "https", will return
// ns/gateway_https
// = Listener Key
func ListenerKey(gw *gatewayv1.Gateway, listener gatewayv1.Listener) client.ObjectKey {
	return ListenerKeyFromListenerName(gw, listener.Name)
}

func ListenerKeyFromListenerName(gw *gatewayv1.Gateway, listenerName gatewayv1.SectionName) client.ObjectKey {
	return client.ObjectKey{
		Namespace: gw.Namespace,
		Name: fmt.Sprintf("%s_%s",
			gw.Name,
			listenerName,
		),
	}
}

// ConvertListenerKeyToGatewayKey converts a listener key back to a gateway key.
// It assumes the listener key is in the format "gateway-name_listener-name", built by the previous ListenerKey function.
// For a listener key with namespace "ns" and name "my-gateway_https",
// it returns a gateway key with namespace "ns" and name "my-gateway".
func ConvertListenerKeyToGatewayKey(listenerKey client.ObjectKey) client.ObjectKey {
	gatewayKey, _, err := ConvertListenerKeyToGatewayKeyAndListenerName(listenerKey)
	if err != nil {
		return listenerKey
	}
	return gatewayKey
}

// ConvertListenerKeyToGatewayKeyAndListenerName converts a listener key back to a gateway key and listener name.
// It assumes the listener key is in the format "gateway-name_listener-name", built by the ListenerKey function.
// For a listener key with namespace "ns" and name "my-gateway_https",
// it returns a gateway key with namespace "ns" and name "my-gateway", the listener name "https", and no error.
// If the format is invalid, it returns an error.
func ConvertListenerKeyToGatewayKeyAndListenerName(listenerKey client.ObjectKey) (client.ObjectKey, string, error) {
	parts := strings.Split(listenerKey.Name, "_")
	if len(parts) != 2 {
		return client.ObjectKey{}, "", fmt.Errorf("invalid listener key format: %s", listenerKey.Name)
	}
	gatewayKey := client.ObjectKey{
		Namespace: listenerKey.Namespace,
		Name:      parts[0],
	}
	return gatewayKey, parts[1], nil
}

func (l *Listener) addAttachedRoute(routeKey client.ObjectKey, controllerStore ControllerStore) {
	l.AttachedRoutes[routeKey] = struct{}{}

	// Find the corresponding Gateway and set it as upserted
	gwKey := l.Owner
	treeGw, ok := controllerStore.GateTree.Gateways[gwKey]
	if !ok || treeGw.TreeStatus.Status == store.StatusDeleted {
		// no action needed, Gateway is Deleted or not manager by our controller
		return
	}

	treeGw.TreeStatus.Status = store.StatusUpserted
	treeGw.TreeStatus.OldTreeResource = treeGw.DeepCopy()
}

func (l *Listener) deleteAttachedRoute(routeKey client.ObjectKey, controllerStore ControllerStore) {
	if l.AttachedRoutes == nil {
		return
	}
	delete(l.AttachedRoutes, routeKey)

	// Find the corresponding Gateway and set it as upserted
	gwKey := l.Owner
	treeGw, ok := controllerStore.GateTree.Gateways[gwKey]
	if !ok || treeGw.TreeStatus.Status == store.StatusDeleted {
		// no action needed, Gateway is Deleted or not manager by our controller
		return
	}

	treeGw.TreeStatus.Status = store.StatusUpserted
}
