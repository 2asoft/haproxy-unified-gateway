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
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Builder interface {
	Build()
	BuildStatus()
}

type BuilderParams struct {
	ClusterStore *store.ClusterStore
	GateTree     *GateTree
	Logger       *slog.Logger
	ExtractGVK   utils.ExtractGVK
}

// GateTree is a Graph-like representation of Gateway API resources.
type GateTree struct {
	// // GatewayClasses holds the GatewayClasses resource that are accepted and ignored
	GatewayClasses CategorizedGatewayClasses
	// ReferencedSecrets includes Secrets referenced by Gateway Listeners, including invalid ones.
	// It is different from the other maps, because it includes entries for Secrets that do not exist
	// in the cluster. We need such entries so that we can query the Graph to determine if a Secret is referenced
	// by the Gateway, including the case when the Secret is newly created.
	Gateways map[types.NamespacedName]*Gateway
	// ReferencedHaproxyGates includes the Gates that are references by GatewayClasses and Gateways
	ReferencedHaproxyGates ReferencedBy
	ReferencedSecrets      map[types.NamespacedName]*Secret
	// ReferencedNamespaces includes Namespaces with labels that match the Gateway Listener's label selector.
	ReferencedNamespaces map[types.NamespacedName]*v1.Namespace
	// ReferencedServices includes the NamespacedNames of all the Services that are referenced by at least one Route.
	ReferencedServices map[types.NamespacedName]*Service
	// A Map of installed GwApi CRDs versions
	InstalledGwAPIVersions InstalledVersions
	IsGwAPIVersionValid    bool
}

func NewGateTree(extractGVK utils.ExtractGVK) *GateTree {
	return &GateTree{
		InstalledGwAPIVersions: InstalledVersions{
			Versions: make(map[string]int),
		},
		GatewayClasses: CategorizedGatewayClasses{
			Supported: make(map[types.NamespacedName]*GatewayClass),
			Ignored:   make(map[types.NamespacedName]*GatewayClass),
		},
		Gateways:               make(map[types.NamespacedName]*Gateway),
		ReferencedHaproxyGates: NewReferencedBy(extractGVK),
		ReferencedSecrets:      make(map[types.NamespacedName]*Secret),
		ReferencedNamespaces:   make(map[types.NamespacedName]*v1.Namespace),
		ReferencedServices:     make(map[types.NamespacedName]*Service),
	}
}

func (t *GateTree) IsSupportedGatewayClass(gwcNsName client.ObjectKey) bool {
	_, ok := t.GatewayClasses.Supported[gwcNsName]
	return ok
}

func (t *GateTree) IsIgnoredGatewayClass(gwcNsName client.ObjectKey) bool {
	_, ok := t.GatewayClasses.Ignored[gwcNsName]
	return ok
}
