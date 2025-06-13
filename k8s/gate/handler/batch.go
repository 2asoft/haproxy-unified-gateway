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
	"time"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/events"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/status"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	v1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// EventHandler handles a batch of events.
// Its builds a ClusterStore of k8s resources.
type EventHandler interface {
	// HandleEventBatch handles a batch of events.
	// EventBatch can include duplicated events.
	HandleEventBatch(ctx context.Context, batch events.EventBatch)
}

type EventHandlerImplConfig struct {
	Logger                   *slog.Logger
	LogCategoryFilterHandler *logging.CategoryFilterHandler
	ExtractGVK               utils.ExtractGVK
	TreeChannel              chan *tree.GateTree
	//  Namespace and name of the controller conf CRD
	ControllerConfNsName types.NamespacedName
}

// eventHandlerImpl implements EventHandler.
// eventHandlerImpl is responsible for:
// - Reconciling the Gateway API and Kubernetes built-in resources with the HAProxy configuration.
// - building the GateTree
type eventHandlerImpl struct {
	treeBuilder *GateTreeBuilder
	config      EventHandlerImplConfig
}

// NewEventHandlerImpl creates a new eventHandlerImpl.
func NewEventHandlerImpl(
	treeBuilderConfig GateTreeBuilderConfig,
	config EventHandlerImplConfig,
) *eventHandlerImpl {
	clusterStore := &store.ClusterStore{
		GatewayClasses:  make(map[types.NamespacedName]*gatewayv1.GatewayClass),
		Gateways:        make(map[types.NamespacedName]*gatewayv1.Gateway),
		HTTPRoutes:      make(map[types.NamespacedName]*gatewayv1.HTTPRoute),
		Services:        make(map[types.NamespacedName]*v1.Service),
		Namespaces:      make(map[types.NamespacedName]*v1.Namespace),
		Secrets:         make(map[types.NamespacedName]*v1.Secret),
		ConfigMaps:      make(map[types.NamespacedName]*v1.ConfigMap),
		GatewayAPICRDs:  make(map[types.NamespacedName]*metav1.PartialObjectMetadata),
		HaproxyGates:    make(map[types.NamespacedName]*v3.HaproxyGate),
		ControllerConfs: make(map[types.NamespacedName]*v3.HaproxyGateCtrlCfg),
		Updates:         store.NewClusterUpdates(),
	}

	currentTree := tree.NewGateTree()

	treeBuilder := NewGateTreeBuilder(
		clusterStore,
		currentTree,
		config.LogCategoryFilterHandler,
		treeBuilderConfig,
		config.Logger,
	)

	handler := &eventHandlerImpl{
		treeBuilder: treeBuilder,
		config:      config,
	}

	return handler
}

func (h *eventHandlerImpl) HandleEventBatch(ctx context.Context, batch events.EventBatch) {
	start := time.Now()

	h.config.Logger.LogAttrs(context.Background(), slog.LevelInfo,
		"Started processing event batch",
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrBatch(batch.BatchID, len(batch.Events)),
	)

	defer func() {
		duration := time.Since(start)
		h.config.Logger.LogAttrs(context.Background(), slog.LevelInfo,
			"Finished processing event batch",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrBatch(batch.BatchID, len(batch.Events)),
			logging.LogAttrDuration(duration),
		)
	}()

	// Process each event in the batch
	_ = h.treeBuilder.ProcessBatch(batch)

	// Build the GateTree
	h.treeBuilder.buildGateTree()
	gatetree := h.treeBuilder.GetTree()
	// // Send the newTree to the TreeChannel if the channel is configured
	if h.config.TreeChannel != nil {
		h.config.TreeChannel <- gatetree
	}

	// START EXAMPLE
	// Below is just an example
	// We list EndpointSlices using the Service Name Index Field we added as an index to the EndpointSlice cache.
	// This allows us to perform a quick lookup of all EndpointSlices for a Service.
	var endpointSliceList discoveryV1.EndpointSliceList
	svcName := "http-echo"
	svcNs := "default"
	err := h.treeBuilder.cfg.k8sClient.List(
		ctx,
		&endpointSliceList,
		client.MatchingFields{index.EndpointSliceServiceNameIndexField: svcName},
		client.InNamespace(svcNs),
	)
	if err != nil {
		h.config.Logger.LogAttrs(context.Background(), slog.LevelError,
			"could not retrieve http-echo endpoints",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrError(err),
		)
	}
	// h.config.Logger.Info(
	// 	fmt.Sprintf("JUST AN EXAMPLE to show cache indexes usage. eps for http-echo svc %v", endpointSliceList))
	// END EXAMPLE

	statusUpdater := status.NewStatusUpdaterImpl(
		status.NewStatusUpdaterConf(
			h.treeBuilder.cfg.k8sClient,
			h.config.ExtractGVK,
			h.config.Logger,
		),
		gatetree.GatewayClasses.Supported,
		gatetree.GatewayClasses.Ignored,
	)

	statusUpdater.UpdateStatus(ctx)

	// h.updateHAProxy(ctx, logger)  //revive:disable:unused-parameters
	// h.updateStatuses(ctx, logger) //revive:disable:unused-parameter
}
