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
	"fmt"
	"slices"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Listener struct {
	// K8sResource is the source resource.
	K8sResource gatewayv1.Listener
	// Final Conditions
	Conditions conditions.Conditions
	// Checks results
	CheckRouteGroupKind CheckResult
	CheckProtocol       CheckResult
	CheckSecret         CheckResult
	// AllowedRouteKinds is the list of allowed route kinds for this listener.
	AllowedRouteKinds []gatewayv1.RouteGroupKind
	// Valid
	Valid bool
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

func (*Listener) isSupportedCertKindGroup(certRef gatewayv1.SecretObjectReference) bool {
	supportedKind := certRef.Kind == nil || *certRef.Kind == "Secret"
	supportedGroup := certRef.Group == nil || *certRef.Group == ""
	return supportedKind && supportedGroup
}

func (l *Listener) BuildConditions(treeGw *Gateway) {
	l.Conditions.MergeOverrideConditions(l.CheckRouteGroupKind.Conditions)
	l.Conditions.MergeOverrideConditions(l.CheckProtocol.Conditions)
	l.Conditions.MergeOverrideConditions(l.CheckSecret.Conditions)

	// Should we process with Haproxy programmation
	shouldProgramm := true

	_, exists := l.Conditions.GetCondition(conditions.ConditionType(gatewayv1.ListenerConditionAccepted))
	if !exists {
		// Accepted = OK
		l.Conditions.MergeOverrideConditions(conditions.NewListenerAcceptedOK())
	} else {
		l.Conditions.MergeOverrideConditions(conditions.NewListenerProgrammedInvalid())
		shouldProgramm = false
	}

	_, exists = l.Conditions.GetCondition(conditions.ConditionType(gatewayv1.ListenerConditionResolvedRefs))
	if !exists {
		// ResolvedRefs = OK
		l.Conditions.MergeOverrideConditions(conditions.NewListenerResolvedRefOK())
	} else {
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
}
