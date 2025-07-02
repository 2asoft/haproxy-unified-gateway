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
package handler

import (
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
)

type GateTreeBuilder struct {
	cfg GateTreeConfig
	tree.ControllerStore
	referenceManager *tree.ReferenceManager
	builder          []tree.Builder
}

func (b *GateTreeBuilder) GetTree() *tree.GateTree {
	return b.GateTree
}

func NewGateTreeBuilder(
	controllerStore tree.ControllerStore,
	cfg GateTreeConfig,
) GateTreeBuilder {
	// --------------
	// Update References
	// --------------
	referenceManager := tree.NewReferenceManager(controllerStore)

	// --------------
	// GatewayClass
	gatewayClassBuilderParams := tree.GatewayClassBuilderParams{
		ControllerStore: controllerStore,
	}
	gatewayClassBuilder := tree.NewGatewayClassBuilder(gatewayClassBuilderParams)

	// --------------
	// Gateway
	gatewayBuilder := tree.NewGatewayBuilder(tree.GatewayBuilderParams{
		ControllerStore: controllerStore,
	})

	treeBuilder := GateTreeBuilder{
		cfg:              cfg,
		referenceManager: referenceManager,
		ControllerStore:  controllerStore,
		builder: []tree.Builder{
			gatewayClassBuilder,
			gatewayBuilder,
		},
	}

	return treeBuilder
}

func (b *GateTreeBuilder) buildGateTree() {
	// --------------
	// Clean TreeUpdates
	// --------------
	for _, builder := range b.builder {
		builder.CleanTreeUpdates()
	}
	b.ControllerStore.CleanInstalledVersionsUpdates()
	// --------------
	// Update the references
	b.referenceManager.UpdateRefences()

	// --------------
	// controllerConf CRD
	controllerConfBuilderParams := tree.ControllerConfBuilderParams{
		ControllerStore:          b.ControllerStore,
		LogCategoryFilterHandler: b.cfg.LogCategoryFilterHandler,
		ControllerConfNsName:     b.cfg.ControllerConfNsName,
	}
	controllerConfBuilder := tree.NewControllerConfBuilder(controllerConfBuilderParams)
	controllerConfBuilder.Build()
	// --------------
	// installed Versions
	installedVersionBuilder := tree.NewInstalledVersionsBuilder(b.ControllerStore)
	installedVersionBuilder.Build()

	for _, builder := range b.builder {
		builder.ComputeTreeUpdates()
	}

	// compute Config Changes
}
