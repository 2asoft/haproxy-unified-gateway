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

	apiv1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
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
}

type GateTreeBuilder struct {
	clusterStoreUpdater  store.ClusterStoreUpdater
	clusterStore         *store.ClusterStore
	currentTree          *tree.GateTree
	isRelevantChangeFunc map[schema.GroupVersionKind]IsRelevantChangeFunc
	logger               *slog.Logger
	cfg                  GateTreeBuilderConfig
}

// IsRelevantChangeFunc is a function that checks if the object has relevant changes.
type IsRelevantChangeFunc func(object client.Object, nsname types.NamespacedName) bool

func NewGateTreeBuilder(
	clusterStore *store.ClusterStore,
	cfg GateTreeBuilderConfig,
	logger *slog.Logger,
) *GateTreeBuilder {
	clusterStoreUpdater := store.NewClusterStoreUpdaterImpl(
		clusterStore,
		cfg.extractGVK,
		logger.WithGroup("clusterStoreUpdater"),
	)
	treeBuilder := GateTreeBuilder{
		clusterStoreUpdater: clusterStoreUpdater,
		clusterStore:        clusterStore,
		cfg:                 cfg,
		logger:              logger,
	}

	hasRelevantChanges := map[schema.GroupVersionKind]IsRelevantChangeFunc{
		cfg.extractGVK(&gatewayv1.GatewayClass{}):    nil,
		cfg.extractGVK(&gatewayv1.Gateway{}):         nil,
		cfg.extractGVK(&gatewayv1.HTTPRoute{}):       nil,
		cfg.extractGVK(&apiv1.Service{}):             treeBuilder.currentTree.IsReferenced,
		cfg.extractGVK(&apiv1.Namespace{}):           treeBuilder.currentTree.IsReferenced,
		cfg.extractGVK(&apiv1.Secret{}):              treeBuilder.currentTree.IsReferenced,
		cfg.extractGVK(&apiv1.ConfigMap{}):           treeBuilder.currentTree.IsReferenced,
		cfg.extractGVK(&discoveryV1.EndpointSlice{}): treeBuilder.currentTree.IsReferenced,
	}
	treeBuilder.isRelevantChangeFunc = hasRelevantChanges

	return &treeBuilder
}

// NewGateTreeBuilderConfig creates a new TreeBuilderConfig.
func NewGateTreeBuilderConfig(
	k8sClient client.Client,
	k8sReader client.Reader,
	gatewayClassNames map[string]struct{},
	extractGVK utils.ExtractGVK,
) GateTreeBuilderConfig {
	eventHandlerConfig := GateTreeBuilderConfig{
		k8sClient:         k8sClient,
		k8sReader:         k8sReader,
		gatewayClassNames: gatewayClassNames,
		extractGVK:        extractGVK,
	}
	return eventHandlerConfig
}

func (b *GateTreeBuilder) ProcessBatch(batch events.EventBatch) bool {
	relevantChanges := false
	for _, e := range batch.Events {
		change := b.updateClusterStore(e, b.logger)
		relevantChanges = relevantChanges || change
	}
	return relevantChanges
}

func (b *GateTreeBuilder) updateClusterStore(event any, logger *slog.Logger) (relevantChanges bool) {
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
		relevantChangeFunc, ok := b.isRelevantChangeFunc[gvk]
		if !ok {
			return true
		}
		if relevantChangeFunc != nil {
			relevantChanges = relevantChangeFunc(obj.Resource, client.ObjectKeyFromObject(obj.Resource))
		}

	case *events.DeleteEvent:
		gvk := b.cfg.extractGVK(obj.Type)

		logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrEventType("delete"),
			logging.LogAttrResource(obj.Type, gvk),
		)

		b.clusterStoreUpdater.Delete(obj.Type, obj.NamespacedName)
		relevantChangeFunc, ok := b.isRelevantChangeFunc[gvk]
		if !ok {
			return true
		}
		if relevantChangeFunc != nil {
			relevantChanges = relevantChangeFunc(obj.Type, obj.NamespacedName)
		}
	}
	return relevantChanges
}

func (b *GateTreeBuilder) buildGateTree() *tree.GateTree {
	newTree := &tree.GateTree{}

	// --------------
	// GatewayClass
	gatewayClassCategorizer := &tree.GatewayClassCategorizerImpl{}

	gatewayClassBuilderParams := tree.GatewayClassBuilderParams{
		ClusterStore: b.clusterStore,
		GcNames:      b.cfg.gatewayClassNames,
		Categorizer:  gatewayClassCategorizer,
		Logger:       b.logger,
	}
	gatewayClassBuilder := tree.NewGatewayClassBuilder(gatewayClassBuilderParams)
	categorizedGatewayClasses := gatewayClassBuilder.Build()
	newTree.GatewayClasses = categorizedGatewayClasses

	// --------------
	// Gateway
	gatewayBuilder := tree.NewGatewayBuilder(tree.GatewayBuilderParams{
		ClusterStore:   b.clusterStore,
		GatewayClasses: categorizedGatewayClasses.Supported,
		Logger:         b.logger,
	})
	newTree.Gateways = gatewayBuilder.Build()

	// -------------------
	// Status compute
	// -------------------
	// This should be called from the lib called
	// with some info on whereas the config was correctly applied
	// or if they are conflicts
	gatewayClassBuilder.BuildStatus()

	return newTree
}
