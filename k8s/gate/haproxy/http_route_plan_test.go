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
package haproxy

import (
	"testing"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/storage/maps"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestHTTPRoutePlanIndexesHostPathMatchesAcrossRouteBoundaries(t *testing.T) {
	listenerKey := "hug-gateways/shared-gateway/http"
	hostname := gatewayv1.Hostname("example.com")
	exact := gatewayv1.PathMatchExact
	prefix := gatewayv1.PathMatchPathPrefix
	challengePath := "/.well-known/acme-challenge/token"
	rootPath := "/"
	appPath := "/app"

	plan := newHTTPRoutePlan()
	plan.addCandidate(httpRouteCandidate{
		ListenerKeyName: listenerKey,
		RouteValueName:  listenerKey + "/apps/redirect",
		Hostnames:       []gatewayv1.Hostname{hostname},
		Match: gatewayv1.HTTPRouteMatch{Path: &gatewayv1.HTTPPathMatch{
			Type:  &prefix,
			Value: &rootPath,
		}},
		Backend: maps.WeightedValue{ValueName: "redirect-backend"},
	})
	plan.addCandidate(httpRouteCandidate{
		ListenerKeyName: listenerKey,
		RouteValueName:  listenerKey + "/apps/app",
		Hostnames:       []gatewayv1.Hostname{hostname},
		Match: gatewayv1.HTTPRouteMatch{Path: &gatewayv1.HTTPPathMatch{
			Type:  &prefix,
			Value: &appPath,
		}},
		Backend: maps.WeightedValue{ValueName: "app-backend"},
	})
	plan.addCandidate(httpRouteCandidate{
		ListenerKeyName: listenerKey,
		RouteValueName:  listenerKey + "/cert-manager/solver",
		Hostnames:       []gatewayv1.Hostname{hostname},
		Match: gatewayv1.HTTPRouteMatch{Path: &gatewayv1.HTTPPathMatch{
			Type:  &exact,
			Value: &challengePath,
		}},
		Backend: maps.WeightedValue{ValueName: "solver-backend"},
	})

	exactHostPathKey := maps.EntryKey{Hostname: listenerKey + "/" + string(hostname), Path: challengePath}
	exactHostPathBackends := plan.listenerHostPathExact[exactHostPathKey]
	if got := exactHostPathBackends["solver-backend"]; got == nil || got.ValueName != "solver-backend" {
		t.Fatalf("exact listener-host-path route = %+v, want solver-backend", exactHostPathBackends)
	}
	if _, ok := exactHostPathBackends["redirect-backend"]; ok {
		t.Fatal("prefix redirect backend must not be indexed in exact listener-host-path map")
	}

	prefixHostPathKey := maps.EntryKey{Hostname: listenerKey + "/" + string(hostname), Path: appPath}
	prefixHostPathBackends := plan.listenerHostPathPrefix[prefixHostPathKey]
	if got := prefixHostPathBackends["app-backend"]; got == nil || got.ValueName != "app-backend" {
		t.Fatalf("prefix listener-host-path route = %+v, want app-backend", prefixHostPathBackends)
	}
	if _, ok := prefixHostPathBackends["solver-backend"]; ok {
		t.Fatal("exact solver backend must not be indexed in prefix listener-host-path map")
	}

	legacyPrefixKey := maps.EntryKey{Hostname: listenerKey + "/apps/redirect", Path: rootPath}
	legacyPrefixBackends := plan.path.prefix[legacyPrefixKey]
	if got := legacyPrefixBackends["redirect-backend"]; got == nil || got.ValueName != "redirect-backend" {
		t.Fatalf("legacy prefix route = %+v, want redirect-backend", legacyPrefixBackends)
	}
}
