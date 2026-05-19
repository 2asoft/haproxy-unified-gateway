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
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/storage/maps"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type httpRoutePlan struct {
	path                   desiredBackendsMaps
	listenerHostPathExact  map[maps.EntryKey]map[string]*maps.WeightedValue
	listenerHostPathPrefix map[maps.EntryKey]map[string]*maps.WeightedValue
}

type httpRouteCandidate struct {
	ListenerKeyName string
	RouteValueName  string
	Hostnames       []gatewayv1.Hostname
	Match           gatewayv1.HTTPRouteMatch
	Backend         maps.WeightedValue
}

func newHTTPRoutePlan() httpRoutePlan {
	return httpRoutePlan{
		path:                   newDesiredBackendsMaps(),
		listenerHostPathExact:  map[maps.EntryKey]map[string]*maps.WeightedValue{},
		listenerHostPathPrefix: map[maps.EntryKey]map[string]*maps.WeightedValue{},
	}
}

func (p *httpRoutePlan) addCandidate(candidate httpRouteCandidate) {
	bucket, _ := p.path.resolveEntry(candidate.RouteValueName, candidate.Match)
	backend := &maps.WeightedValue{ValueName: candidate.Backend.ValueName, Weight: candidate.Backend.Weight}
	existing := bucket[candidate.Backend.ValueName]
	if existing == nil {
		bucket[candidate.Backend.ValueName] = backend
	} else {
		newWeight := utils.PointerDefaultValueIfNil(existing.Weight) + utils.PointerDefaultValueIfNil(candidate.Backend.Weight)
		existing.Weight = &newWeight
	}

	p.addExactListenerHostPathCandidate(candidate)
	p.addPrefixListenerHostPathCandidate(candidate)
}

func (p *httpRoutePlan) addExactListenerHostPathCandidate(candidate httpRouteCandidate) {
	if candidate.Match.Path == nil || candidate.Match.Path.Type == nil || *candidate.Match.Path.Type != gatewayv1.PathMatchExact || candidate.Match.Path.Value == nil {
		return
	}
	p.addListenerHostPathCandidate(p.listenerHostPathExact, candidate)
}

func (p *httpRoutePlan) addPrefixListenerHostPathCandidate(candidate httpRouteCandidate) {
	if candidate.Match.Path == nil || candidate.Match.Path.Type == nil || *candidate.Match.Path.Type != gatewayv1.PathMatchPathPrefix || candidate.Match.Path.Value == nil {
		return
	}
	p.addListenerHostPathCandidate(p.listenerHostPathPrefix, candidate)
}

func (p *httpRoutePlan) addListenerHostPathCandidate(entries map[maps.EntryKey]map[string]*maps.WeightedValue, candidate httpRouteCandidate) {
	for _, hostname := range candidate.Hostnames {
		key := maps.EntryKey{Hostname: candidate.ListenerKeyName + "/" + string(hostname), Path: *candidate.Match.Path.Value}
		if entries[key] == nil {
			entries[key] = map[string]*maps.WeightedValue{}
		}
		entries[key][candidate.Backend.ValueName] = &maps.WeightedValue{ValueName: candidate.Backend.ValueName, Weight: candidate.Backend.Weight}
	}
}
