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
	"maps"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

type Condition struct {
	Type               ConditionType
	Status             metav1.ConditionStatus
	Reason             string
	Message            string
	ObservedGeneration int64
}

type Conditions map[ConditionType]Condition

func (c Conditions) MergeOverrideConditions(b Conditions) {
	maps.Copy(c, b)
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

func (c Conditions) SetGeneration(generation int64) {
	for _, condition := range c {
		condition.ObservedGeneration = generation
		c[condition.Type] = condition
	}
}

func NewConditionsFromMetav1Conditions(conditions []metav1.Condition) Conditions {
	conditionsMap := make(Conditions)
	for _, condition := range conditions {
		conditionsMap[ConditionType(condition.Type)] = Condition{
			Type:               ConditionType(condition.Type),
			Status:             condition.Status,
			Reason:             condition.Reason,
			Message:            condition.Message,
			ObservedGeneration: condition.ObservedGeneration,
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
			ObservedGeneration: condition.ObservedGeneration,
			LastTransitionTime: now,
		})
	}
	return conditions
}

func (c Conditions) GetCondition(conditionType ConditionType) (Condition, bool) {
	condition, exists := c[conditionType]
	return condition, exists
}

func (c Conditions) GetMessage(conditionType ConditionType) string {
	condition, exists := c.GetCondition(conditionType)
	if !exists {
		return ""
	}
	return condition.Message
}
