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

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HaproxyGate struct {
	// K8sResource is the source resource.
	K8sResource *v3.HaproxyGate
	// Conditions include Conditions for the HaproxyGate.
	Conditions conditions.Conditions
	// Valid shows whether the HaproxyGate is valid.
	Valid bool
}

type HaproxyGateBuilder interface {
	Build() *HaproxyGate
}

type HaproxyGateBuilderImpl struct {
	clusterStore   *store.ClusterStore
	gatewayclasses map[types.NamespacedName]*GatewayClass
	logger         *slog.Logger
}

type HaproxyGateBuilderParams struct {
	ClusterStore   *store.ClusterStore
	GatewayClasses map[types.NamespacedName]*GatewayClass
	Logger         *slog.Logger
}

func NewHaproxyGateBuilder(
	params HaproxyGateBuilderParams,
) HaproxyGateBuilder {
	return &HaproxyGateBuilderImpl{
		clusterStore:   params.ClusterStore,
		gatewayclasses: params.GatewayClasses,
		logger:         params.Logger,
	}
}

func (h *HaproxyGateBuilderImpl) Build() *HaproxyGate {
	hg := &HaproxyGate{}

	_ = h.FindFirstGatewayClassHavingParametersRef()

	return hg
}

// FindFirstGatewayClassHavingParametersRef picks the first (sortied by CreationTimestamp) GatewayClasses
// that has a reference to a configuration.
func (h *HaproxyGateBuilderImpl) FindFirstGatewayClassHavingParametersRef() *GatewayClass {
	gatewayClassesWithReferencedGateways := make(map[types.NamespacedName]*GatewayClass)
	for _, gc := range h.gatewayclasses {
		if gc.K8sResource.Spec.ParametersRef != nil {
			gatewayClassesWithReferencedGateways[client.ObjectKeyFromObject(gc.K8sResource)] = gc
		}
	}
	if len(gatewayClassesWithReferencedGateways) == 0 {
		return nil
	}

	sortedGc := utils.MapToSortedListByCreationTimestamp(gatewayClassesWithReferencedGateways)

	return sortedGc[0]
}
