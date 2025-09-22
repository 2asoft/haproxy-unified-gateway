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
package routeconditions

import (
	genericconditions "github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ RouteConditionAccessor[*gatewayv1.HTTPRoute] = &HTTPRouteConditionImpl{}

type HTTPRouteConditionImpl struct {
	ControllerName string
}

func (r *HTTPRouteConditionImpl) GetConditions(obj *gatewayv1.HTTPRoute) RouteConditions {
	return NewRouteConditionsFromRouteConditions(obj.Status.RouteStatus, r.ControllerName)
}

func (*HTTPRouteConditionImpl) SetConditions(obj *gatewayv1.HTTPRoute, conds RouteConditions) {
	// TODO preserve other conditions from other controllers
	obj.Status.RouteStatus = conds.ToRouteConditions()
}

func ConditionAccepted() genericconditions.Conditions {
	return genericconditions.Conditions{
		genericconditions.ConditionType(gatewayv1.RouteConditionAccepted): {
			Type:    genericconditions.ConditionType(gatewayv1.RouteConditionAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.RouteReasonAccepted),
			Message: "Route Accepted",
		},
	}
}

func ConditionNotAcceptedNoMatchingParent() genericconditions.Conditions {
	return genericconditions.Conditions{
		genericconditions.ConditionType(gatewayv1.RouteConditionAccepted): {
			Type:    genericconditions.ConditionType(gatewayv1.RouteConditionAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.RouteReasonNoMatchingParent),
			Message: "no matching parent found",
		},
	}
}

func ConditionNotAcceptedNoMatchingHostname() genericconditions.Conditions {
	return genericconditions.Conditions{
		genericconditions.ConditionType(gatewayv1.RouteConditionAccepted): {
			Type:    genericconditions.ConditionType(gatewayv1.RouteConditionAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.RouteReasonNoMatchingListenerHostname),
			Message: "no matching hostname found",
		},
	}
}

func ConditionRouteReasonNotAllowedByListeners() genericconditions.Conditions {
	return genericconditions.Conditions{
		genericconditions.ConditionType(gatewayv1.RouteConditionAccepted): {
			Type:    genericconditions.ConditionType(gatewayv1.RouteConditionAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.RouteReasonNotAllowedByListeners),
			Message: "route kind not allowed by listeners",
		},
	}
}
