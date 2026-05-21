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
package tree

import (
	"testing"
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/conditions/generic"
	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func exampleSupportedKinds() map[gatewayv1.ProtocolType][]gatewayv1.RouteGroupKind {
	return map[gatewayv1.ProtocolType][]gatewayv1.RouteGroupKind{
		gatewayv1.HTTPProtocolType:  {objtypes.RouteKindHTTP, objtypes.RouteKindGRPC},
		gatewayv1.HTTPSProtocolType: {objtypes.RouteKindHTTP, objtypes.RouteKindGRPC},
		gatewayv1.TLSProtocolType:   {objtypes.RouteKindTLS, objtypes.RouteKindTCP},
		gatewayv1.UDPProtocolType:   {objtypes.RouteKindUDP},
		gatewayv1.TCPProtocolType:   {objtypes.RouteKindTCP},
	}
}

func TestSupportedKinds(t *testing.T) {
	tests := []struct {
		name          string
		listener      gatewayv1.Listener
		expectedKinds []gatewayv1.RouteGroupKind
	}{
		{
			name: "HTTP protocol, no allowed kinds specified",
			listener: gatewayv1.Listener{
				Protocol: gatewayv1.HTTPProtocolType,
			},
			expectedKinds: []gatewayv1.RouteGroupKind{objtypes.RouteKindGRPC, objtypes.RouteKindHTTP},
		},
		{
			name: "TLS protocol, no allowed kinds specified",
			listener: gatewayv1.Listener{
				Protocol: gatewayv1.TLSProtocolType,
			},
			expectedKinds: []gatewayv1.RouteGroupKind{objtypes.RouteKindTCP, objtypes.RouteKindTLS},
		},
		{
			name: "HTTP protocol, empty allowed kinds",
			listener: gatewayv1.Listener{
				Protocol: gatewayv1.HTTPProtocolType,
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Kinds: []gatewayv1.RouteGroupKind{},
				},
			},
			expectedKinds: []gatewayv1.RouteGroupKind{objtypes.RouteKindGRPC, objtypes.RouteKindHTTP},
		},
		{
			name: "HTTP protocol, nil allowed kinds",
			listener: gatewayv1.Listener{
				Protocol: gatewayv1.HTTPProtocolType,
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Kinds: nil,
				},
			},
			expectedKinds: []gatewayv1.RouteGroupKind{objtypes.RouteKindGRPC, objtypes.RouteKindHTTP},
		},
		{
			name: "HTTP protocol, with allowed kinds (intersection)",
			listener: gatewayv1.Listener{
				Protocol: gatewayv1.HTTPProtocolType,
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Kinds: []gatewayv1.RouteGroupKind{
						objtypes.RouteKindHTTP, // supported
						objtypes.RouteKindTCP,  // not supported for HTTP
					},
				},
			},
			expectedKinds: []gatewayv1.RouteGroupKind{objtypes.RouteKindHTTP},
		},
		{
			name: "HTTP protocol, with allowed kinds, none supported",
			listener: gatewayv1.Listener{
				Protocol: gatewayv1.HTTPProtocolType,
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Kinds: []gatewayv1.RouteGroupKind{
						objtypes.RouteKindTLS,
					},
				},
			},
			expectedKinds: []gatewayv1.RouteGroupKind{},
		},
		{
			name: "Unsupported protocol",
			listener: gatewayv1.Listener{
				Protocol: "foo",
			},
			expectedKinds: []gatewayv1.RouteGroupKind{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := supportedKinds(tt.listener, exampleSupportedKinds())
			assert.Equal(t, tt.expectedKinds, result)
		})
	}
}

func Test_hostnameConflicts(t *testing.T) {
	tests := []struct {
		name      string
		hostname1 string
		hostname2 string
		want      bool
	}{
		{
			name:      "exact match",
			hostname1: "example.com",
			hostname2: "example.com",
			want:      true,
		},
		{
			name:      "case-insensitive match",
			hostname1: "Example.com",
			hostname2: "example.com",
			want:      true,
		},
		{
			name:      "wildcards are identical",
			hostname1: "*.example.com",
			hostname2: "*.example.com",
			want:      true,
		},
		{
			name:      "both empty - two catch-all listeners conflict",
			hostname1: "",
			hostname2: "",
			want:      true,
		},
		// The following pairs are different hostname values: specificity rules (exact >
		// wildcard > empty) always produce an unambiguous winner, so no conflict.
		{
			name:      "wildcard vs matching specific - no conflict (specific beats wildcard)",
			hostname1: "*.example.com",
			hostname2: "foo.example.com",
			want:      false,
		},
		{
			name:      "specific vs matching wildcard - no conflict (specific beats wildcard)",
			hostname1: "foo.example.com",
			hostname2: "*.example.com",
			want:      false,
		},
		{
			name:      "wildcard vs parent domain - no conflict",
			hostname1: "*.example.com",
			hostname2: "example.com",
			want:      false,
		},
		{
			name:      "two different wildcards - no conflict",
			hostname1: "*.foo.com",
			hostname2: "*.bar.com",
			want:      false,
		},
		{
			name:      "two different exact hostnames - no conflict",
			hostname1: "one.com",
			hostname2: "two.com",
			want:      false,
		},
		{
			name:      "empty vs specific - no conflict (empty coexists with specific)",
			hostname1: "",
			hostname2: "one.com",
			want:      false,
		},
		{
			name:      "specific vs empty - no conflict (empty coexists with specific)",
			hostname1: "one.com",
			hostname2: "",
			want:      false,
		},
		{
			name:      "empty vs wildcard - no conflict (empty coexists with wildcard)",
			hostname1: "",
			hostname2: "*.example.com",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostnameConflicts(tt.hostname1, tt.hostname2); got != tt.want {
				t.Errorf("hostnameConflicts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListenerBuildConditionsDoesNotPreserveStaleInvalidProgrammedWhenChecksBecomeValid(t *testing.T) {
	gatewayCreatedAt := metav1.NewTime(time.Now().Add(-time.Hour))
	gatewayGeneration := int64(7)
	listenerName := gatewayv1.SectionName("rtc-https")
	programmedType := generic.ConditionType(gatewayv1.ListenerConditionProgrammed)

	gw := &Gateway{K8sResource: &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "hug-gateways",
			Name:              "shared-gateway",
			Generation:        gatewayGeneration,
			CreationTimestamp: gatewayCreatedAt,
		},
		Status: gatewayv1.GatewayStatus{Listeners: []gatewayv1.ListenerStatus{{
			Name: listenerName,
			Conditions: []metav1.Condition{{
				Type:               string(gatewayv1.ListenerConditionProgrammed),
				Status:             metav1.ConditionFalse,
				Reason:             string(gatewayv1.ListenerReasonInvalid),
				Message:            "Listener is invalid",
				ObservedGeneration: gatewayGeneration,
				LastTransitionTime: metav1.NewTime(time.Now()),
			}},
		}}},
	}}
	listener := &Listener{
		Owner: client.ObjectKey{Namespace: "hug-gateways", Name: "shared-gateway"},
		K8sResource: gatewayv1.Listener{
			Name:     listenerName,
			Protocol: gatewayv1.HTTPSProtocolType,
			Port:     443,
		},
		Conditions:          make(generic.Conditions),
		CheckRouteGroupKind: CheckResult{Valid: true},
		CheckProtocol:       CheckResult{Valid: true},
		CheckSecret:         CheckResult{Valid: true},
		CheckConflict:       CheckResult{Valid: true},
	}

	listener.BuildConditions(gw)

	programmed, ok := listener.Conditions.GetCondition(programmedType)
	if !ok {
		t.Fatal("Programmed condition not found")
	}
	if programmed.Status != metav1.ConditionUnknown || programmed.Reason != string(gatewayv1.ListenerReasonPending) {
		t.Fatalf("Programmed = %s/%s, want Unknown/Pending after listener checks became valid", programmed.Status, programmed.Reason)
	}
	if !listener.Valid {
		t.Fatal("listener should be valid after checks pass")
	}
}

func TestListenerBuildConditionsPreservesSameGenerationProgrammedTrue(t *testing.T) {
	gatewayCreatedAt := metav1.NewTime(time.Now().Add(-time.Hour))
	gatewayGeneration := int64(8)
	listenerName := gatewayv1.SectionName("https")
	programmedType := generic.ConditionType(gatewayv1.ListenerConditionProgrammed)

	gw := &Gateway{K8sResource: &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "hug-gateways",
			Name:              "shared-gateway",
			Generation:        gatewayGeneration,
			CreationTimestamp: gatewayCreatedAt,
		},
		Status: gatewayv1.GatewayStatus{Listeners: []gatewayv1.ListenerStatus{{
			Name: listenerName,
			Conditions: []metav1.Condition{{
				Type:               string(gatewayv1.ListenerConditionProgrammed),
				Status:             metav1.ConditionTrue,
				Reason:             string(gatewayv1.ListenerReasonProgrammed),
				Message:            "Listener is programmed in Haproxy",
				ObservedGeneration: gatewayGeneration,
				LastTransitionTime: metav1.NewTime(time.Now()),
			}},
		}}},
	}}
	listener := &Listener{
		Owner: client.ObjectKey{Namespace: "hug-gateways", Name: "shared-gateway"},
		K8sResource: gatewayv1.Listener{
			Name:     listenerName,
			Protocol: gatewayv1.HTTPSProtocolType,
			Port:     443,
		},
		Conditions:          make(generic.Conditions),
		CheckRouteGroupKind: CheckResult{Valid: true},
		CheckProtocol:       CheckResult{Valid: true},
		CheckSecret:         CheckResult{Valid: true},
		CheckConflict:       CheckResult{Valid: true},
	}

	listener.BuildConditions(gw)

	programmed, ok := listener.Conditions.GetCondition(programmedType)
	if !ok {
		t.Fatal("Programmed condition not found")
	}
	if programmed.Status != metav1.ConditionTrue || programmed.Reason != string(gatewayv1.ListenerReasonProgrammed) {
		t.Fatalf("Programmed = %s/%s, want True/Programmed", programmed.Status, programmed.Reason)
	}
}
