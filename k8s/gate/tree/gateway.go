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
	"k8s.io/apimachinery/pkg/types"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Gateway represents the Gateway resource.
type Gateway struct {
	// K8sResource is the source resource.
	K8sResource *v1.Gateway
	// Conditions include Conditions for the GatewayClass.
	Conditions conditions.Conditions
	// HaproxyGate contains the HaproxyGate (confguration CRD)
	// The HaproxyGate can be defined at the GatewayClass level or at the Gateway level.
	// Here it is the merged HaproxyGate between the GatewayClass and the Gateway.
	HaproxyGate *v3.HaproxyGate
	// Valid shows whether the GatewayClass is valid.
	Valid bool
}

var _ Builder = &GatewayBuilderImpl{}

type GatewayBuilderImpl struct {
	clusterStore   *store.ClusterStore
	gatewayclasses map[types.NamespacedName]*GatewayClass
	logger         *slog.Logger
}

type GatewayBuilderParams struct {
	// ClusterStore is the store of k8s resources.
	ClusterStore *store.ClusterStore
	// GatewayClasses is a map of GatewayClass resources (only accepted by the controller).
	GatewayClasses map[types.NamespacedName]*GatewayClass
	// Logger is the logger for the GatewayBuilder.
	Logger *slog.Logger
}

func NewGatewayBuilder(params GatewayBuilderParams) *GatewayBuilderImpl {
	return &GatewayBuilderImpl{
		clusterStore:   params.ClusterStore,
		gatewayclasses: params.GatewayClasses,
		logger:         params.Logger,
	}
}

func (*GatewayBuilderImpl) Build() {
	// _ := b.FilterGatewaysByGatewayClass()
	// do all checks...
	// merge HaproxyGate from GatewayClass and Gateway
	// compute Conditions, Status, Valid
}

func (b *GatewayBuilderImpl) FilterGatewaysByGatewayClass() map[types.NamespacedName]*Gateway {
	gateways := make(map[types.NamespacedName]*Gateway)
	for _, gateway := range b.clusterStore.Gateways {
		gatewayClassNsName := types.NamespacedName{Name: string(gateway.Spec.GatewayClassName)}
		if _, ok := b.gatewayclasses[gatewayClassNsName]; !ok {
			continue
		}
		gatewayNsName := types.NamespacedName{Namespace: gateway.Namespace, Name: gateway.Name}
		gateways[gatewayNsName] = &Gateway{K8sResource: gateway}
	}
	return gateways
}

func (*GatewayBuilderImpl) BuildStatus() {
}
