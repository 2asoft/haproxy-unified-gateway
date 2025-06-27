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
	"context"
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/events"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GateTreeBuilderConfig holds configuration parameters for the Gate Tree builder.
type GateTreeBuilderConfig struct {
	// k8sClient is a Kubernetes API client.
	k8sClient client.Client
	// k8sReader is a Kubernets API reader.
	k8sReader client.Reader
	// extractGVK is a function that extracts the GroupVersionKind (GVK) of a client.object.
	extractGVK utils.ExtractGVK
	// gatewayClassName is the name of the supported GatewayClass.
	// If empty, all GatewayClasses are supported that match the controller name
	gatewayClassNames map[string]struct{}
	logger            *slog.Logger
	// ControllerConfNsName is the namespace and name of the controller configuration CRD.
	ControllerConfNsName types.NamespacedName
}

type GateTreeBuilder struct {
	clusterStoreUpdater   store.ClusterStoreUpdater
	clusterStore          *store.ClusterStore
	categoryFilterHandler *logging.CategoryFilterHandler
	tree                  *tree.GateTree
	cfg                   GateTreeBuilderConfig
	builders              []tree.Builder
}

func (b *GateTreeBuilder) GetTree() *tree.GateTree {
	return b.tree
}

func NewGateTreeBuilder(
	clusterStore *store.ClusterStore,
	gateTree *tree.GateTree,
	categoryFilterHandler *logging.CategoryFilterHandler,
	cfg GateTreeBuilderConfig,
	logger *slog.Logger,
) *GateTreeBuilder {
	clusterStoreUpdater := store.NewClusterStoreUpdaterImpl(
		clusterStore,
		cfg.extractGVK,
		logger.WithGroup("clusterStoreUpdater"),
	)

	builderParams := tree.BuilderParams{
		ClusterStore: clusterStore,
		GateTree:     gateTree,
		ExtractGVK:   cfg.extractGVK,
		Logger:       logger,
	}

	// --------------
	// controllerConf
	controllerConfBuilderParams := tree.ControllerConfBuilderParams{
		BuilderParams:            builderParams,
		LogCategoryFilterHandler: categoryFilterHandler,
		ControllerConfNsName:     cfg.ControllerConfNsName,
	}
	controllerConfBuilder := tree.NewControllerConfBuilder(controllerConfBuilderParams)

	// --------------
	// GatewayClass
	gatewayClassBuilderParams := tree.GatewayClassBuilderParams{
		BuilderParams: builderParams,
		GcNames:       cfg.gatewayClassNames,
	}
	gatewayClassBuilder := tree.NewGatewayClassBuilder(gatewayClassBuilderParams)

	// --------------
	// Gateway
	gatewayBuilder := tree.NewGatewayBuilder(tree.GatewayBuilderParams{
		BuilderParams: builderParams,
	})

	treeBuilder := GateTreeBuilder{
		clusterStoreUpdater:   clusterStoreUpdater,
		clusterStore:          clusterStore,
		categoryFilterHandler: categoryFilterHandler,
		cfg:                   cfg,
		tree:                  gateTree,
		builders: []tree.Builder{
			controllerConfBuilder,
			gatewayClassBuilder,
			gatewayBuilder,
		},
	}

	return &treeBuilder
}

// NewGateTreeBuilderConfig creates a new TreeBuilderConfig.
func NewGateTreeBuilderConfig(
	k8sClient client.Client,
	k8sReader client.Reader,
	controllerConfNsName types.NamespacedName,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) GateTreeBuilderConfig {
	eventHandlerConfig := GateTreeBuilderConfig{
		k8sClient:            k8sClient,
		k8sReader:            k8sReader,
		ControllerConfNsName: controllerConfNsName,
		extractGVK:           extractGVK,
		logger:               logger,
	}
	return eventHandlerConfig
}

func (b *GateTreeBuilder) ProcessBatch(batch events.EventBatch) bool {
	b.clusterStoreUpdater.ResetUpdates()

	for _, e := range batch.Events {
		b.updateClusterStore(e, b.cfg.logger)
	}
	return true
}

func (b *GateTreeBuilder) updateClusterStore(event any, logger *slog.Logger) {
	switch obj := event.(type) {
	case *events.UpsertEvent:
		gvk := b.cfg.extractGVK(obj.Resource)
		logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrEventType("upsert"),
			logging.LogAttrResource(obj.Resource, gvk),
		)

		b.clusterStoreUpdater.Upsert(obj.Resource)

	case *events.DeleteEvent:
		gvk := b.cfg.extractGVK(obj.Type)

		logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrEventType("delete"),
			logging.LogAttrResource(obj.Type, gvk),
		)

		b.clusterStoreUpdater.Delete(obj.Type, obj.NamespacedName)
	}
}

func (b *GateTreeBuilder) buildGateTree() {
	for _, builder := range b.builders {
		builder.Build()
	}

	// -------------------
	// Status compute
	// -------------------
	// This should be called from the lib
	// with some info on whereas the config was correctly applied
	// or if they are conflicts
	for _, builder := range b.builders {
		builder.BuildStatus()
	}
}
