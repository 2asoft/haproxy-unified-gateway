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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ ConditionAccessor[*v1.GatewayClass] = &GatewayClassConditionImpl{}

type GatewayClassConditionImpl struct{}

func (*GatewayClassConditionImpl) GetConditions(obj *v1.GatewayClass) Conditions {
	return NewConditionsFromMetav1Conditions(obj.Status.Conditions)
}

func (*GatewayClassConditionImpl) SetConditions(obj *v1.GatewayClass, conds Conditions) {
	obj.Status.Conditions = conds.ToMetav1Conditions()
}

// NewGatewayClassUnsupportedVersion returns Conditions to indicate:
// - the Gateway API CRD versions are not supported.
// Only applies to Accepted GatewayClasses
// Ignored GatewayClasses will have a Conflict Condition
func NewGatewayClassUnsupportedVersion(recommendedVersion string) map[ConditionType]Condition {
	return map[ConditionType]Condition{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:   ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status: metav1.ConditionTrue,
			Reason: string(v1.GatewayClassReasonUnsupportedVersion),
			Message: fmt.Sprintf(
				"Gateway API CRD versions are not supported. Best effort. Please install version %s",
				recommendedVersion,
			),
		},
		ConditionType(v1.GatewayClassConditionStatusSupportedVersion): {
			Type:   ConditionType(v1.GatewayClassConditionStatusSupportedVersion),
			Status: metav1.ConditionFalse,
			Reason: string(v1.GatewayClassReasonUnsupportedVersion),
			Message: fmt.Sprintf(
				"Gateway API CRD versions are not supported. Please install version %s",
				recommendedVersion,
			),
		},
	}
}

// NewGatewayClassConflict returns a Condition that indicates that the GatewayClass is not accepted
// due to a conflict with another GatewayClass.
func NewGatewayClassConflict() map[ConditionType]Condition {
	return map[ConditionType]Condition{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(GatewayClassReasonGatewayClassConflict),
			Message: GatewayClassMessageGatewayClassConflict,
		},
	}
}

// NewDefaultGatewayClassConditions returns Conditions that indicate that the GatewayClass is accepted and that the
// Gateway API CRD versions are supported.
func NewDefaultGatewayClassConditions() map[ConditionType]Condition {
	return map[ConditionType]Condition{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(v1.GatewayClassReasonAccepted),
			Message: "GatewayClass is accepted",
		},
		ConditionType(v1.GatewayClassConditionStatusSupportedVersion): {
			Type:    ConditionType(v1.GatewayClassConditionStatusSupportedVersion),
			Status:  metav1.ConditionTrue,
			Reason:  string(v1.GatewayClassReasonSupportedVersion),
			Message: "Gateway API CRD versions are supported",
		},
	}
}
