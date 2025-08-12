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

var _ ConditionAccessor[*v1.Gateway] = &GatewayConditionImpl{}

type GatewayConditionImpl struct{}

func (*GatewayConditionImpl) GetConditions(obj *v1.Gateway) Conditions {
	return NewConditionsFromMetav1Conditions(obj.Status.Conditions)
}

func (*GatewayConditionImpl) SetConditions(obj *v1.Gateway, conds Conditions) {
	obj.Status.Conditions = conds.ToMetav1Conditions()
}

// ---------------------------------------------------------
// GatewayConditionAccepted

func NewGatewayAcceptedOK() Conditions {
	return Conditions{
		ConditionType(v1.GatewayConditionAccepted): {
			Type:    ConditionType(v1.GatewayConditionAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(v1.GatewayReasonAccepted),
			Message: "Gateway is accepted",
		},
	}
}

func NewGatewayAcceptedInvalidConditions(msg string) Conditions {
	return Conditions{
		ConditionType(v1.GatewayConditionAccepted): {
			Type:    ConditionType(v1.GatewayConditionAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(v1.GatewayReasonInvalid),
			Message: fmt.Sprintf("GatewayClass '%s' is not accepted", msg),
		},
	}
}

func NewGatewayAcceptedInvalidParameters(err *field.Error) Conditions {
	return Conditions{
		ConditionType(v1.GatewayConditionAccepted): {
			Type:    ConditionType(v1.GatewayReasonInvalidParameters),
			Status:  metav1.ConditionFalse,
			Reason:  string(v1.GatewayReasonInvalidParameters),
			Message: fmt.Sprintf("invalid parametersRef: %s", err),
		},
	}
}

// ---------------------------------------------------------
// GatewayConditionProgrammed

func NewGatewayProgrammedOK() Conditions {
	return Conditions{
		ConditionType(v1.GatewayConditionProgrammed): {
			Type:    ConditionType(v1.GatewayConditionProgrammed),
			Status:  metav1.ConditionTrue,
			Reason:  string(v1.GatewayConditionProgrammed),
			Message: "Gateway is programmed",
		},
	}
}

func NewGatewayProgrammedInvalidParameters(msg string) Conditions {
	return Conditions{
		ConditionType(v1.GatewayConditionProgrammed): {
			Type:    ConditionType(v1.GatewayConditionProgrammed),
			Status:  metav1.ConditionFalse,
			Reason:  string(v1.GatewayReasonInvalidParameters),
			Message: msg,
		},
	}
}
