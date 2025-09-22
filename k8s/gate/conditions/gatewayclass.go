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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// This reason is used with GatewayClassConditionAccepted (false).
	GatewayClassReasonGatewayClassConflict gatewayv1.GatewayClassConditionReason = "GatewayClassConflict"

	// GatewayClassMessageGatewayClassConflict is a message that describes GatewayClassReasonGatewayClassConflict.
	GatewayClassMessageGatewayClassConflict = "Resource ignored due to a conflicting GatewayClass resource"

	// GatewayClassReasonUnsupported is a message that describes GatewayClassReasonUnsupported
	GatewayClassReasonUnsupported = "Resource ignored due to an unsupported GatewayClass"
)

var _ generic.ConditionAccessor[*gatewayv1.GatewayClass] = &GatewayClassConditionImpl{}

type GatewayClassConditionImpl struct{}

func (*GatewayClassConditionImpl) GetConditions(obj *gatewayv1.GatewayClass) generic.Conditions {
	return generic.NewConditionsFromMetav1Conditions(obj.Status.Conditions)
}

func (*GatewayClassConditionImpl) SetConditions(obj *gatewayv1.GatewayClass, conds generic.Conditions) {
	obj.Status.Conditions = conds.ToMetav1Conditions()
}

// NewDefaultGatewayClassConditions returns genericconditions.Conditions that indicate that the GatewayClass is accepted and that the
// Gateway API CRD versions are supported.
func NewDefaultGatewayClassConditions() generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.GatewayClassReasonAccepted),
			Message: "GatewayClass is accepted",
		},
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusSupportedVersion): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusSupportedVersion),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.GatewayClassReasonSupportedVersion),
			Message: "Gateway API CRD versions are supported",
		},
	}
}

// ---------------------------------------------------------
// GatewayClassConditionStatusAccepted

func NewGatewayClassAcceptedOK() generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.GatewayClassReasonAccepted),
			Message: "GatewayClass is accepted",
		},
	}
}

func NewGatewayClassAcceptedConflict() generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(GatewayClassReasonGatewayClassConflict),
			Message: GatewayClassMessageGatewayClassConflict,
		},
	}
}

func NewGatewayClassAcceptedUnsupported() generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.GatewayClassReasonUnsupported),
			Message: GatewayClassReasonUnsupported,
		},
	}
}

// NewGatewayClassAcceptedInvalidParameters returns a Condition that indicates that the GatewayClass has invalid parameters.
func NewGatewayClassAcceptedInvalidParameters(err *field.Error) generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.GatewayClassReasonInvalidParameters),
			Message: fmt.Sprintf("invalid parametersRef: %s", err),
		},
	}
}

// ---------------------------------------------------------
// GatewayClassConditionStatusSupportedVersion

// NewGatewayClassSupportedVersionUnsupportedVersion returns genericconditions.Conditions to indicate:
// - the Gateway API CRD versions are not supported.
func NewGatewayClassSupportedVersionUnsupportedVersion(recommendedVersion string) generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusSupportedVersion): {
			Type:   generic.ConditionType(gatewayv1.GatewayClassConditionStatusSupportedVersion),
			Status: metav1.ConditionFalse,
			Reason: string(gatewayv1.GatewayClassReasonUnsupportedVersion),
			Message: fmt.Sprintf(
				"Gateway API CRD versions are not supported. Please install version %s",
				recommendedVersion,
			),
		},
	}
}

func NewGatewayClassSupportedVersionOK() generic.Conditions {
	return generic.Conditions{
		generic.ConditionType(gatewayv1.GatewayClassConditionStatusSupportedVersion): {
			Type:    generic.ConditionType(gatewayv1.GatewayClassConditionStatusSupportedVersion),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.GatewayClassReasonSupportedVersion),
			Message: "Gateway API CRD versions are supported",
		},
	}
}
