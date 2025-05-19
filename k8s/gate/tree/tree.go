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
	v1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Graph is a Graph-like representation of Gateway API resources.
type GateTree struct {
	// // GatewayClasses holds the GatewayClasses resource that are accepted and ignored
	GatewayClasses CategorizedGatewayClasses
	// ReferencedSecrets includes Secrets referenced by Gateway Listeners, including invalid ones.
	// It is different from the other maps, because it includes entries for Secrets that do not exist
	// in the cluster. We need such entries so that we can query the Graph to determine if a Secret is referenced
	// by the Gateway, including the case when the Secret is newly created.
	ReferencedSecrets map[types.NamespacedName]*Secret
	// ReferencedNamespaces includes Namespaces with labels that match the Gateway Listener's label selector.
	ReferencedNamespaces map[types.NamespacedName]*v1.Namespace
	// ReferencedServices includes the NamespacedNames of all the Services that are referenced by at least one Route.
	ReferencedServices map[types.NamespacedName]*Service
}

// IsReferenced returns true if the Graph references the resource.
func (g *GateTree) IsReferenced(resourceType client.Object, nsname types.NamespacedName) bool {
	if g == nil {
		return false
	}
	// switch obj := resourceType.(type) {
	switch resourceType.(type) {
	case *v1.Secret:
		// Check if secret is a Gateway-referenced Secret
		_, exists := g.ReferencedSecrets[nsname]
		return exists
	case *v1.Namespace:
		// HELENE: implement this
		exists := true
		return exists
	// Service reference exists if at least one HTTPRoute references it.
	case *v1.Service:
		_, exists := g.ReferencedServices[nsname]
		return exists
	// EndpointSlice reference exists if its Service owner is referenced by at least one HTTPRoute.
	case *discoveryV1.EndpointSlice:
		// HELENE implement this
		exists := true
		return exists
	default:
		return false
	}
}
