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
	"maps"

	generic "github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type RouteConditions struct {
	Conditions     map[gatewayv1.ParentReference]map[generic.ConditionType]generic.Condition
	ControllerName string
}

func (c RouteConditions) MergeOverrideConditions(b RouteConditions) {
	maps.Copy(c.Conditions, b.Conditions)
}

func (c RouteConditions) Equal(b RouteConditions) bool {
	if c.ControllerName != b.ControllerName {
		return false
	}
	if len(c.Conditions) != len(b.Conditions) {
		return false
	}
	for k, v := range c.Conditions {
		if len(v) != len(b.Conditions[k]) {
			return false
		}

		for k2, v2 := range v {
			if b.Conditions[k][k2] != v2 {
				return false
			}
		}
	}
	return true
}

func (c RouteConditions) SetGeneration(generation int64) {
	for parentRef, mapConditions := range c.Conditions {
		for _, condition := range mapConditions {
			condition.ObservedGeneration = generation
			c.Conditions[parentRef][condition.Type] = condition
		}
	}
}

func NewRouteConditionsFromRouteConditions(routeStatus gatewayv1.RouteStatus, controllerName string) RouteConditions {
	// map[gatewayv1.ParentReference]map[ConditionType]Condition
	conditionsMap := make(map[gatewayv1.ParentReference]map[generic.ConditionType]generic.Condition)
	for _, parentConditions := range routeStatus.Parents {
		parentRef := parentConditions.ParentRef
		if parentConditions.ControllerName != gatewayv1.GatewayController(controllerName) {
			continue
		}

		if len(conditionsMap[parentRef]) == 0 {
			conditionsMap[parentRef] = make(map[generic.ConditionType]generic.Condition)
		}

		for _, condition := range parentConditions.Conditions {
			conditionsMap[parentRef][generic.ConditionType(condition.Type)] = generic.Condition{
				Type:               generic.ConditionType(condition.Type),
				Status:             condition.Status,
				Reason:             condition.Reason,
				Message:            condition.Message,
				ObservedGeneration: condition.ObservedGeneration,
			}
		}
	}
	return RouteConditions{
		Conditions:     conditionsMap,
		ControllerName: controllerName,
	}
}

func (c RouteConditions) ToRouteConditions() gatewayv1.RouteStatus {
	now := metav1.Now()
	result := gatewayv1.RouteStatus{}
	// map[gatewayv1.ParentReference]map[ConditionType]Condition
	for parentRef, conditions := range c.Conditions {
		for conditionType, condition := range conditions {
			result.Parents = append(result.Parents, gatewayv1.RouteParentStatus{
				ParentRef:      parentRef,
				ControllerName: gatewayv1.GatewayController(c.ControllerName),
				Conditions: []metav1.Condition{
					{
						Type:               string(conditionType),
						Status:             condition.Status,
						Reason:             condition.Reason,
						Message:            condition.Message,
						ObservedGeneration: condition.ObservedGeneration,
						LastTransitionTime: now,
					},
				},
			})
		}
	}
	return result
}

// GetCondition of certain type
func (c RouteConditions) GetCondition(conditionType generic.ConditionType, parentRef gatewayv1.ParentReference) (generic.Condition, bool) {
	parent, exists := c.Conditions[parentRef]
	if !exists {
		return generic.Condition{}, false
	}

	condition, exists := parent[conditionType]
	return condition, exists
}
