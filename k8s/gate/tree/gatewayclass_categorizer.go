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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayClassCategorizerImpl struct {
	gcNames  map[string]struct{}
	gateTree *GateTree
}

// Supported contains the GatewayClasses that are accepted
// Ignored holds the ignored GatewayClass resources, which reference Haproxy Gateway API controller in
// // `.spec.controllerName`,
// // Those GatewayClass are needed as GatewayAPI spec
// // This is used to update the status of the those GatewayClass resources
type CategorizedGatewayClasses struct {
	Supported map[types.NamespacedName]*GatewayClass
	Ignored   map[types.NamespacedName]*GatewayClass
}

type GatewayClassCategorizer interface {
	Categorize(map[types.NamespacedName]store.Update[*gatewayv1.GatewayClass])
}

// CategorizedK8sGatewayClasses is a struct that contains the categorized GatewayClass resources.
// It contains two maps:
// - Supported: GatewayClass resources that are supported by the controller.
// - Ignored: GatewayClass resources that are ignored by the controller.
func (c *GatewayClassCategorizerImpl) Categorize(gcUpdates map[types.NamespacedName]store.Update[*gatewayv1.GatewayClass]) {
	for nsName, gc := range gcUpdates {
		switch gc.Status {
		case store.StatusUpserted:
			c.processUpserted(gc)
		case store.StatusDeleted:
			c.processDeleted(nsName)
		}
	}
}

func (c *GatewayClassCategorizerImpl) processUpserted(gcUpdate store.Update[*gatewayv1.GatewayClass]) {
	gc := gcUpdate.NewObject
	_, allowedGcName := c.gcNames[gc.Name]
	treeGc := GatewayClass{
		K8sResource: gc,
	}

	if allowedGcName {
		c.gateTree.GatewayClasses.Supported[client.ObjectKeyFromObject(gc)] = &treeGc
	} else {
		c.gateTree.GatewayClasses.Ignored[client.ObjectKeyFromObject(gc)] = &treeGc
	}
}

func (c *GatewayClassCategorizerImpl) processDeleted(nsName types.NamespacedName) {
	delete(c.gateTree.GatewayClasses.Supported, nsName)
	delete(c.gateTree.GatewayClasses.Ignored, nsName)
}
