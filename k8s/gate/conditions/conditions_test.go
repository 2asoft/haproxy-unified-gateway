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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestMergeOverrideConditions(t *testing.T) {
	// Initial base conditions
	a := Conditions{
		ConditionType(v1.GatewayClassConditionStatusAccepted): Condition{
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionFalse,
			Reason:  "resaon1",
			Message: "msg1",
		},
		ConditionType(v1.GatewayClassConditionStatusSupportedVersion): Condition{
			Type:    ConditionType(v1.GatewayClassConditionStatusSupportedVersion),
			Status:  metav1.ConditionTrue,
			Reason:  "reason2",
			Message: "msg2",
		},
	}

	// Incoming override conditions
	b := Conditions{
		ConditionType(v1.GatewayClassConditionStatusAccepted): Condition{
			Type:    ConditionType(v1.GatewayClassConditionStatusAccepted),
			Status:  metav1.ConditionTrue,
			Reason:  "reason1-2",
			Message: "msg1-2",
		},
		ConditionType("new-type"): Condition{
			Type:    ConditionType("new-type"),
			Status:  metav1.ConditionTrue,
			Reason:  "reason3",
			Message: "msg3",
		},
	}

	// Merge b into a
	a.MergeOverrideConditions(b)
	// Check total number of keys
	if len(a) != 3 {
		t.Errorf("expected 3 conditions, got %d", len(a))
	}

	// Check overridden
	c := a[ConditionType(v1.GatewayClassConditionStatusAccepted)]
	if c.Status != metav1.ConditionTrue || c.Reason != "reason1-2" || c.Message != "msg1-2" {
		t.Errorf("override failed for 'Accepted': got %+v", c)
	}

	// Check unchanged
	c = a[ConditionType(v1.GatewayClassConditionStatusSupportedVersion)]
	if c.Status != metav1.ConditionTrue || c.Reason != "reason2" || c.Message != "msg2" {
		t.Errorf("unexpected change for 'SupportedVersion': got %+v", c)
	}

	// Check newly added
	c = a[ConditionType("new-type")]
	if c.Status != metav1.ConditionTrue || c.Reason != "reason3" || c.Message != "msg3" {
		t.Errorf("missing or incorrect new condition: got %+v", c)
	}
}

func TestConditions_Equal(t *testing.T) {
	tests := []struct {
		name     string
		a        Conditions
		b        Conditions
		expected bool
	}{
		{
			name:     "empty conditions",
			a:        Conditions{},
			b:        Conditions{},
			expected: true,
		},
		{
			name: "equal conditions",
			a: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			b: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			expected: true,
		},
		{
			name: "different number of conditions",
			a: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
			},
			b: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			expected: false,
		},
		{
			name: "different condition",
			a: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			b: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionTrue, Reason: "reason2", Message: "message2"},
			},
			expected: false,
		},
		{
			name: "different reason",
			a: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			b: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reasonnot1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			expected: false,
		},
		{
			name: "different messages",
			a: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reason1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "message2"},
			},
			b: Conditions{
				"type1": {Type: "type1", Status: metav1.ConditionTrue, Reason: "reasonn1", Message: "message1"},
				"type2": {Type: "type2", Status: metav1.ConditionFalse, Reason: "reason2", Message: "messagenot2"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		if result := tt.a.Equal(tt.b); result != tt.expected {
			t.Errorf("Test case %q failed: expected %v, got %v", tt.name, tt.expected, result)
		}
	}
}
