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

	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"
	"github.com/stretchr/testify/assert"
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

func Test_matchesWithWildcard(t *testing.T) {
	tests := []struct {
		name     string
		wildcard string
		hostname string
		want     bool
	}{
		{
			name:     "subdomain",
			wildcard: "*.example.com",
			hostname: "foo.example.com",
			want:     true,
		},
		{
			name:     "not a subdomain, just the domain",
			wildcard: "*.example.com",
			hostname: "example.com",
			want:     false, // Gateway API spec: A wildcard domain does not match the parent domain
		},
		{
			name:     "not a subdomain, just the suffix",
			wildcard: "*.example.com",
			hostname: ".example.com",
			want:     false,
		},
		{
			name:     "non-matching domain",
			wildcard: "*.example.com",
			hostname: "foo.example.org",
			want:     false,
		},
		{
			name:     "empty hostname",
			wildcard: "*.example.com",
			hostname: "",
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesWithWildcard(tt.wildcard, tt.hostname); got != tt.want {
				t.Errorf("matchesWildcard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_overlaps(t *testing.T) {
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
			name:      "wildcard h1 matches h2",
			hostname1: "*.example.com",
			hostname2: "foo.example.com",
			want:      true,
		},
		{
			name:      "wildcard h2 matches h1",
			hostname1: "foo.example.com",
			hostname2: "*.example.com",
			want:      true,
		},
		{
			name:      "wildcard does not match parent domain",
			hostname1: "*.example.com",
			hostname2: "example.com",
			want:      false, // Gateway API spec: A wildcard domain does not match the parent domain
		},
		{
			name:      "wildcards are identical",
			hostname1: "*.example.com",
			hostname2: "*.example.com",
			want:      true,
		},
		{
			name:      "wildcards do not overlap",
			hostname1: "*.foo.com",
			hostname2: "*.bar.com",
			want:      false,
		},
		{
			name:      "no match",
			hostname1: "one.com",
			hostname2: "two.com",
			want:      false,
		},
		{
			name:      "empty hostname #1",
			hostname1: "",
			hostname2: "one.com",
			want:      true,
		},
		{
			name:      "empty hostname #2",
			hostname1: "one.com",
			hostname2: "",
			want:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := overlaps(tt.hostname1, tt.hostname2); got != tt.want {
				t.Errorf("overlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}
