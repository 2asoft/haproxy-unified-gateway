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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ---------------------------------------------------------
// ListenerConditionResolvedRefs

func NewListenerResolvedRefInvalidRouteKinds(msg string) Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionResolvedRefs): {
			Type:    ConditionType(gatewayv1.ListenerConditionResolvedRefs),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.ListenerReasonInvalidRouteKinds),
			Message: msg,
		},
	}
}

func NewListenerResolvedRefOK() Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionResolvedRefs): {
			Type:    ConditionType(gatewayv1.ListenerConditionResolvedRefs),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.ListenerReasonResolvedRefs),
			Message: "Listener references have been resolved",
		},
	}
}

// ---------------------------------------------------------
// ListenerConditionProgrammed

func NewListenerProgrammedPending() Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionProgrammed): {
			Type:    ConditionType(gatewayv1.ListenerConditionProgrammed),
			Status:  metav1.ConditionUnknown,
			Reason:  string(gatewayv1.ListenerReasonPending),
			Message: "Listener is pending Haproxy programmation",
		},
	}
}

func NewListenerProgrammedInvalid() Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionProgrammed): {
			Type:    ConditionType(gatewayv1.ListenerConditionProgrammed),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.ListenerReasonInvalid),
			Message: "Listener is invalid",
		},
	}
}

func NewListenerProgrammedOK() Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionProgrammed): {
			Type:    ConditionType(gatewayv1.ListenerConditionProgrammed),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.ListenerReasonProgrammed),
			Message: "Listener is programmed in Haproxy",
		},
	}
}

// ---------------------------------------------------------
// ListenerConditionAccepted

func NewListenerAcceptedUnsupportedProtocol(msg string) Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionAccepted): {
			Type:    ConditionType(gatewayv1.ListenerConditionAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  string(gatewayv1.ListenerReasonUnsupportedProtocol),
			Message: msg,
		},
	}
}

func NewListenerAcceptedOK() Conditions {
	return Conditions{
		ConditionType(gatewayv1.ListenerConditionAccepted): {
			Type:    ConditionType(gatewayv1.ListenerConditionAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  string(gatewayv1.ListenerReasonAccepted),
			Message: "Listener is accepted",
		},
	}
}
