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
	"k8s.io/apimachinery/pkg/util/validation/field"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// This reason is used with GatewayClassConditionAccepted (false).
	GatewayClassReasonGatewayClassConflict v1.GatewayClassConditionReason = "GatewayClassConflict"

	// GatewayClassMessageGatewayClassConflict is a message that describes GatewayClassReasonGatewayClassConflict.
	GatewayClassMessageGatewayClassConflict = "Resource ignored due to a conflicting GatewayClass resource"

	// GatewayClassReasonUnsupported is a message that describes GatewayClassReasonUnsupported
	GatewayClassReasonUnsupported = "Resource ignored due to an unsupported GatewayClass"
)

var _ ConditionAccessor[*v1.GatewayClass] = &GatewayClassConditionImpl{}

type GatewayClassConditionImpl struct{}

func (*GatewayClassConditionImpl) GetConditions(obj *v1.GatewayClass) Conditions {
	return NewConditionsFromMetav1Conditions(obj.Status.Conditions)
}

func (*GatewayClassConditionImpl) SetConditions(obj *v1.GatewayClass, conds Conditions) {
	obj.Status.Conditions = conds.ToMetav1Conditions()
}

// NewDefaultGatewayClassConditions returns Conditions that indicate that the GatewayClass is accepted and that the
// Gateway API CRD versions are supported.
func NewDefaultGatewayClassConditions() Conditions {
	return Conditions{
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

// ---------------------------------------------------------
// GatewayClassConditionStatusAccepted

func NewGatewayClassAcceptedOK() Conditions {
	return Conditions{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(v1.GatewayClassReasonAccepted),
			Message: "GatewayClass is accepted",
		},
	}
}

func NewGatewayClassAcceptedConflict() Conditions {
	return Conditions{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(GatewayClassReasonGatewayClassConflict),
			Message: GatewayClassMessageGatewayClassConflict,
		},
	}
}

func NewGatewayClassAcceptedUnsupported() Conditions {
	return Conditions{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(v1.GatewayClassReasonUnsupported),
			Message: GatewayClassReasonUnsupported,
		},
	}
}

// NewGatewayClassAcceptedInvalidParameters returns a Condition that indicates that the GatewayClass has invalid parameters.
func NewGatewayClassAcceptedInvalidParameters(err *field.Error) Conditions {
	return Conditions{
		ConditionType(v1.GatewayClassConditionStatusAccepted): {
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(v1.GatewayClassReasonInvalidParameters),
			Message: fmt.Sprintf("invalid parametersRef: %s", err),
		},
	}
}

// ---------------------------------------------------------
// GatewayClassConditionStatusSupportedVersion

// NewGatewayClassSupportedVersionUnsupportedVersion returns Conditions to indicate:
// - the Gateway API CRD versions are not supported.
func NewGatewayClassSupportedVersionUnsupportedVersion(recommendedVersion string) Conditions {
	return Conditions{
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

func NewGatewayClassSupportedVersionOK() Conditions {
	return Conditions{
		ConditionType(v1.GatewayClassConditionStatusSupportedVersion): {
			Type:    ConditionType(v1.GatewayClassConditionStatusSupportedVersion),
			Status:  metav1.ConditionTrue,
			Reason:  string(v1.GatewayClassReasonSupportedVersion),
			Message: "Gateway API CRD versions are supported",
		},
	}
}
