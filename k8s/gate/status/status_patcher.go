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
package status

import (
	"fmt"
	"slices"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type StatusPatcher interface {
	StatusEqual(client.Object) (bool, error)
	SetStatus(client.Object) error
}

// ---------------------------
// GatewayClass
func newGatewayClassStatusPatcher(gwc *tree.GatewayClass) StatusPatcher {
	return &gatewayClassStatusPatcher{
		conditions: gwc.Conditions,
	}
}

var _ StatusPatcher = &gatewayClassStatusPatcher{}

type gatewayClassStatusPatcher struct {
	conditions generic.Conditions
}

func (sm *gatewayClassStatusPatcher) StatusEqual(obj client.Object) (bool, error) {
	gwc, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		return false, fmt.Errorf("wrong type %T", obj)
	}
	conds := generic.NewConditionsFromMetav1Conditions(gwc.Status.Conditions)
	return sm.conditions.Equal(conds), nil
}

func (sm *gatewayClassStatusPatcher) SetStatus(obj client.Object) error {
	gwc, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		return fmt.Errorf("wrong type %T", obj)
	}
	metav1conds := sm.conditions.ToMetav1Conditions()
	gwc.Status = gatewayv1.GatewayClassStatus{
		Conditions: metav1conds,
	}
	return nil
}

// ---------------------------
// Gateway
func newGatewayStatusPatcher(gw *tree.Gateway) StatusPatcher {
	listenerStatuses := make([]gatewayv1.ListenerStatus, 0, len(gw.Listeners))
	if gw.Valid {
		// If the Gateway is invalid, the listeners status should be empty
		for _, listener := range gw.Listeners {
			conds := listener.Conditions.ToMetav1Conditions()
			listenerStatuses = append(listenerStatuses, gatewayv1.ListenerStatus{
				Name:           listener.K8sResource.Name,
				Conditions:     conds,
				SupportedKinds: listener.AllowedRouteKinds,
				AttachedRoutes: 0, // To be changed with the correct value
			})
		}
	}

	return &gatewayStatusPatcher{
		conditions:       gw.Conditions,
		listenerStatuses: listenerStatuses,
	}
}

var _ StatusPatcher = &gatewayStatusPatcher{}

type gatewayStatusPatcher struct {
	conditions       generic.Conditions
	listenerStatuses []gatewayv1.ListenerStatus
}

func (sm *gatewayStatusPatcher) StatusEqual(obj client.Object) (bool, error) {
	gw, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		return false, fmt.Errorf("wrong type %T", obj)
	}
	gwConds := generic.NewConditionsFromMetav1Conditions(gw.Status.Conditions)
	if !sm.conditions.Equal(gwConds) {
		return false, nil
	}

	return ListenerStatusesEqual(sm.listenerStatuses, gw.Status.Listeners), nil
}

func ListenerStatusesEqual(a, b []gatewayv1.ListenerStatus) bool {
	sortListenerStatusByName(a)
	sortListenerStatusByName(b)
	listenerStatusEqual := func(a, b gatewayv1.ListenerStatus) bool {
		if a.Name != b.Name {
			return false
		}

		if a.AttachedRoutes != b.AttachedRoutes {
			return false
		}

		aConds := generic.NewConditionsFromMetav1Conditions(a.Conditions)
		bConds := generic.NewConditionsFromMetav1Conditions(b.Conditions)
		if !aConds.Equal(bConds) {
			return false
		}

		return cmp.Equal(a.SupportedKinds, b.SupportedKinds)
	}
	return slices.EqualFunc(a, b, listenerStatusEqual)
}

func (sm *gatewayStatusPatcher) SetStatus(obj client.Object) error {
	gw, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		return fmt.Errorf("wrong type %T", obj)
	}
	metav1conds := sm.conditions.ToMetav1Conditions()

	gw.Status = gatewayv1.GatewayStatus{
		Conditions: metav1conds,
		Listeners:  sm.listenerStatuses,
	}
	return nil
}

// sortListenerStatusByName sorts a slice of gatewayapi.ListenerStatus structs
// in place, in ascending order based on the `Name` field.
// It uses the standard library's `slices.SortFunc` for efficient sorting.
func sortListenerStatusByName(listeners []gatewayv1.ListenerStatus) {
	slices.SortFunc(listeners, func(a, b gatewayv1.ListenerStatus) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
}
