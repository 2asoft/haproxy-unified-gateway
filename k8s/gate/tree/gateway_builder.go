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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

var _ Builder = &GatewayBuilderImpl{}

type GatewayBuilderImpl struct {
	BuilderParams
}

type GatewayBuilderParams struct {
	BuilderParams
}

func NewGatewayBuilder(params GatewayBuilderParams) *GatewayBuilderImpl {
	return &GatewayBuilderImpl{
		BuilderParams: params.BuilderParams,
	}
}

func (b *GatewayBuilderImpl) Build() {
	// Gateway updates
	b.OnGatewaysUpdated()
}

func (b *GatewayBuilderImpl) OnGatewaysUpdated() {
	for _, gwUpdate := range b.ClusterStore.Updates.Gateways {
		gw, ok := gwUpdate.GetObject().(*v1.Gateway)
		if !ok {
			continue
		}
		switch gwUpdate.Status {
		case store.StatusUpserted:
			b.OnGatewayUpserted(gw)
		case store.StatusDeleted:
		}
	}
}

func (b *GatewayBuilderImpl) OnGatewayUpserted(gw *v1.Gateway) {
	if b.isGatewayClassAccepted(gw) {
		treeGw := NewGateway(gw)
		b.GateTree.Gateways[client.ObjectKeyFromObject(gw)] = treeGw
	}
}

func (b *GatewayBuilderImpl) OnGatewayDeleted(gw *v1.Gateway) {
	delete(b.GateTree.Gateways, client.ObjectKeyFromObject(gw))
}

func (b *GatewayBuilderImpl) isGatewayClassAccepted(gateway *v1.Gateway) bool {
	// Is GatewayClass part of the accepted GatewayClasses
	gwKey := client.ObjectKeyFromObject(gateway)
	if _, ok := b.GateTree.GatewayClasses.Supported[gwKey]; !ok {
		return false
	}
	return true
}

func (*GatewayBuilderImpl) BuildStatus() {
}
