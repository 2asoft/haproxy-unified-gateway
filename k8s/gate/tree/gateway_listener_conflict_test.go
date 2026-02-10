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

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/protocols"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestComputeListenerConflicts(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	_ = t2

	// Helper to create a Gateway
	mkGateway := func(name string, creationTime time.Time, listeners ...gatewayv1.Listener) *Gateway {
		gw := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         "default",
				CreationTimestamp: metav1.NewTime(creationTime),
			},
			Spec: gatewayv1.GatewaySpec{
				Listeners: listeners,
			},
		}
		return &Gateway{
			K8sResource: gw,
		}
	}

	// Helper to create a Listener
	mkListener := func(name string, port int, protocol gatewayv1.ProtocolType, hostname *string) gatewayv1.Listener {
		var h *gatewayv1.Hostname
		if hostname != nil {
			val := gatewayv1.Hostname(*hostname)
			h = &val
		}
		return gatewayv1.Listener{
			Name:     gatewayv1.SectionName(name),
			Port:     gatewayv1.PortNumber(port),
			Protocol: protocol,
			Hostname: h,
		}
	}

	ptr := func(s string) *string { return &s }
	_ = ptr

	tests := []struct {
		name     string
		gateways []*Gateway
		expected map[string]listenerConflictCondition
	}{
		{
			name: "No conflicts - single gateway, different ports",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, nil),
					mkListener("l2", 443, gatewayv1.HTTPSProtocolType, nil),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_l1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_l2": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategorySecure},
			},
		},
		{
			name: "Conflict - same port, different protocol categories",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("http", 80, gatewayv1.HTTPProtocolType, nil),
					mkListener("https", 80, gatewayv1.HTTPSProtocolType, nil), // 80 is already HTTP/Insecure
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_http":  {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_https": {hasConflict: true, reason: string(gatewayv1.ListenerReasonProtocolConflict), protocol: protocols.ProtocolCategorySecure},
			},
		},
		{
			name: "No Conflict - same port, same protocol category (Insecure), non-overlapping hostnames",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("h1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
					mkListener("h2", 80, gatewayv1.HTTPProtocolType, ptr("bar.com")),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_h1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_h2": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Conflict - same port, same protocol category, overlapping hostnames (Exact)",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("h1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
					mkListener("h2", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_h1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_h2": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Conflict - same port, overlapping hostnames (Wildcard)",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("wild", 80, gatewayv1.HTTPProtocolType, ptr("*.example.com")),
					mkListener("exact", 80, gatewayv1.HTTPProtocolType, ptr("foo.example.com")),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_wild":  {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_exact": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Conflict - same port, overlapping hostnames (empty)",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("empty", 80, gatewayv1.HTTPProtocolType, nil),
					mkListener("same", 80, gatewayv1.HTTPProtocolType, nil),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_empty": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_same":  {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Multiple Gateways - oldest wins",
			gateways: []*Gateway{
				mkGateway("old", t1,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
				mkGateway("new", t2,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/old_l1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/new_l1": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Multiple Gateways - oldest wins (reverse order in input)",
			gateways: []*Gateway{
				mkGateway("new", t2,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
				mkGateway("old", t1,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/old_l1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/new_l1": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Protocol conflict across gateways",
			gateways: []*Gateway{
				mkGateway("old", t1,
					mkListener("http", 80, gatewayv1.HTTPProtocolType, nil),
				),
				mkGateway("new", t2,
					mkListener("https", 80, gatewayv1.HTTPSProtocolType, nil),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/old_http":  {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/new_https": {hasConflict: true, reason: string(gatewayv1.ListenerReasonProtocolConflict), protocol: protocols.ProtocolCategorySecure},
			},
		},
		{
			name: "Complex scenario",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("http", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
					mkListener("http2", 80, gatewayv1.HTTPProtocolType, ptr("bar.com")),
				),
				mkGateway("gw2", t2,
					mkListener("conflict-foo", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
					mkListener("conflict-all", 80, gatewayv1.HTTPProtocolType, nil), // nil = matches all, overlaps with everything
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_http":         {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_http2":        {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw2_conflict-foo": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
				"default/gw2_conflict-all": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Mix of ProtocolTypes and Hostnames on same port",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("http-foo", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),   // Winner (Insecure)
					mkListener("https-bar", 80, gatewayv1.HTTPSProtocolType, ptr("bar.com")), // Conflict (Secure != Insecure, winner is Insecure), distinct host irrelevant
					mkListener("tls-foo", 80, gatewayv1.TLSProtocolType, ptr("foo.com")),     // Conflict (TLS != Insecure, winner is Insecure), overlapping host irrelevant
					mkListener("http-wild", 80, gatewayv1.HTTPProtocolType, ptr("*.com")),    // Same category (Insecure). Overlaps "foo.com". Conflict.
					mkListener("http-bar", 80, gatewayv1.HTTPProtocolType, ptr("bar.com")),   // Same category. No overlap with "foo.com". OK.
					mkListener("tcp", 80, gatewayv1.TCPProtocolType, nil),                    // Conflict (Unknown != Insecure)
				),
				mkGateway("gw2", t1,
					mkListener("tls-foo", 443, gatewayv1.TLSProtocolType, ptr("foo.com")),     // Winner (TLS)
					mkListener("https-bar", 443, gatewayv1.HTTPSProtocolType, ptr("bar.com")), // Conflict (Secure != TLS)
					mkListener("tls-wild", 443, gatewayv1.TLSProtocolType, ptr("*.com")),      // Conflict (Overlaps "foo.com")
					mkListener("tls-bar", 443, gatewayv1.TLSProtocolType, ptr("bar.com")),     // OK (No overlap with "foo.com")
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_http-foo":  {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_https-bar": {hasConflict: true, reason: string(gatewayv1.ListenerReasonProtocolConflict), protocol: protocols.ProtocolCategorySecure},
				"default/gw1_tls-foo":   {hasConflict: true, reason: string(gatewayv1.ListenerReasonProtocolConflict), protocol: protocols.ProtocolCategoryTLS},
				"default/gw1_http-wild": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_http-bar":  {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw1_tcp":       {hasConflict: true, reason: string(gatewayv1.ListenerReasonProtocolConflict), protocol: protocols.ProcotolCategoryTCP},
				"default/gw2_tls-foo":   {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryTLS},
				"default/gw2_https-bar": {hasConflict: true, reason: string(gatewayv1.ListenerReasonProtocolConflict), protocol: protocols.ProtocolCategorySecure},
				"default/gw2_tls-wild":  {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryTLS},
				"default/gw2_tls-bar":   {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryTLS},
			},
		},
		{
			name: "Nil gateways (DELETED status) should be ignored",
			gateways: []*Gateway{
				mkGateway("gw1", t1,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
				nil, // Deleted gateway should be ignored
				mkGateway("gw2", t2,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("bar.com")),
				),
				nil, // Another deleted gateway
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_l1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw2_l1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
			},
		},
		{
			name: "Nil gateway with conflict - only non-nil gateways participate",
			gateways: []*Gateway{
				nil, // Deleted gateway should not claim the port
				mkGateway("gw1", t1,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
				mkGateway("gw2", t2,
					mkListener("l1", 80, gatewayv1.HTTPProtocolType, ptr("foo.com")),
				),
			},
			expected: map[string]listenerConflictCondition{
				"default/gw1_l1": {hasConflict: false, reason: "", protocol: protocols.ProtocolCategoryInsecure},
				"default/gw2_l1": {hasConflict: true, reason: string(gatewayv1.ListenerReasonHostnameConflict), protocol: protocols.ProtocolCategoryInsecure},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &GatewayBuilderImpl{
				ControllerStore: &ControllerStore{
					GateTree: &GateTree{
						Gateways: make(map[client.ObjectKey]*Gateway),
					},
				},
			}
			b.resetListenerConflicts()

			// Populate store
			for _, gw := range tt.gateways {
				var key client.ObjectKey
				if gw != nil {
					key = client.ObjectKeyFromObject(gw.K8sResource)
				}
				b.GateTree.Gateways[key] = gw
			}

			b.computeListenerConflicts()

			// Flatten results
			actual := make(map[string]listenerConflictCondition)

			for _, res := range b.ControllerStore.mapPort2Listeners {
				for k, v := range res {
					actual[k.String()] = v
				}
			}

			// Verify results
			assert.Equal(t, len(tt.expected), len(actual), "Unexpected number of results")

			for k, expectedVal := range tt.expected {
				actualVal, found := actual[k]
				assert.True(t, found, "Expected result for %s not found", k)
				assert.Equal(t, expectedVal, actualVal, "Mismatch for %s", k)
			}
		})
	}
}
