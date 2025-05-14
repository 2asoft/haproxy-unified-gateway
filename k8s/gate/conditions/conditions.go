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
package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// This reason is used with GatewayClassConditionAccepted (false).
	GatewayClassReasonGatewayClassConflict v1.GatewayClassConditionReason = "GatewayClassConflict"

	// GatewayClassMessageGatewayClassConflict is a message that describes GatewayClassReasonGatewayClassConflict.
	GatewayClassMessageGatewayClassConflict = "Resource ignored due to a conflicting GatewayClass resource"
)

type ConditionType string

type Condition struct {
	Type    ConditionType
	Status  metav1.ConditionStatus
	Reason  string
	Message string
}

type Conditions map[ConditionType]Condition

func (c Conditions) MergeOverrideConditions(b Conditions) {
	for k, v := range b {
		c[k] = v
	}
}

func (c Conditions) Equal(b Conditions) bool {
	if len(c) != len(b) {
		return false
	}
	for k, v := range c {
		if b[k] != v {
			return false
		}
	}
	return true
}

func NewConditionsFromMetav1Conditions(conditions []metav1.Condition) Conditions {
	conditionsMap := make(Conditions)
	for _, condition := range conditions {
		conditionsMap[ConditionType(condition.Type)] = Condition{
			Type:    ConditionType(condition.Type),
			Status:  condition.Status,
			Reason:  condition.Reason,
			Message: condition.Message,
		}
	}
	return conditionsMap
}

func (c Conditions) ToMetav1Conditions() []metav1.Condition {
	now := metav1.Now()
	conditions := make([]metav1.Condition, 0, len(c))
	for _, condition := range c {
		conditions = append(conditions, metav1.Condition{
			Type:               string(condition.Type),
			Status:             condition.Status,
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: now,
		})
	}
	return conditions
}
