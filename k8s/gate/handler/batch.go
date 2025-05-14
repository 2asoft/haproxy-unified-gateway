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
	"fmt"
	"log/slog"
	"time"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/events"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/status"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
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

// eventHandlerImpl implements EventHandler.
// eventHandlerImpl is responsible for:
// - Reconciling the Gateway API and Kubernetes built-in resources with the HAProxy configuration.
// - building the GateTree
type eventHandlerImpl struct {
	treeBuilder *GateTreeBuilder
	logger      *slog.Logger

	extractGVK utils.ExtractGVK
}

// NewEventHandlerImpl creates a new eventHandlerImpl.
func NewEventHandlerImpl(
	treeBuilderConfig GateTreeBuilderConfig,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) *eventHandlerImpl {
	clusterStore := &store.ClusterStore{
		GatewayClasses: make(map[types.NamespacedName]*gatewayv1.GatewayClass),
		Gateways:       make(map[types.NamespacedName]*gatewayv1.Gateway),
		HTTPRoutes:     make(map[types.NamespacedName]*gatewayv1.HTTPRoute),
		Services:       make(map[types.NamespacedName]*v1.Service),
		Namespaces:     make(map[types.NamespacedName]*v1.Namespace),
		Secrets:        make(map[types.NamespacedName]*v1.Secret),
		ConfigMaps:     make(map[types.NamespacedName]*v1.ConfigMap),
		GatewayAPICRDs: make(map[types.NamespacedName]*metav1.PartialObjectMetadata),
	}

	treeBuilder := NewGateTreeBuilder(
		clusterStore,
		treeBuilderConfig,
		logger.WithGroup("treeBuilder"),
	)

	handler := &eventHandlerImpl{
		treeBuilder: treeBuilder,
		extractGVK:  extractGVK,
		logger:      logger,
	}

	return handler
}

func (h *eventHandlerImpl) HandleEventBatch(ctx context.Context, batch events.EventBatch) {
	start := time.Now()
	h.logger.With("batchID", batch.BatchID).Info("Started processing event batch", "len", len(batch.Events))

	defer func() {
		duration := time.Since(start)
		h.logger.With("batchID", batch.BatchID).With("len", len(batch.Events)).Info(
			"Finished processing event batch",
			"duration", duration.String(),
		)
	}()

	// Process each event in the batch
	hasRelevantChanges := h.treeBuilder.ProcessBatch(batch)
	h.logger.Info("HELENE", "hasRelevantChanges", hasRelevantChanges)

	// Build the GateTree
	newTree := h.treeBuilder.buildGateTree()
	h.logger.Info("HELENE", "newTree", newTree)

	// Update some sort of store
	// Compute config
	// Compute diffs....
	// What we need to do has to be done

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
		h.logger.Error("error client.List svc http-echo", "error", err)
	}
	h.logger.Info(
		fmt.Sprintf("JUST AN EXAMPLE to show cache indexes usage. endpoints for http-echo svc %v", endpointSliceList))
	// END EXAMPLE

	statusUpdater := status.NewStatusUpdaterImpl(
		status.NewStatusUpdaterConf(
			h.treeBuilder.cfg.k8sClient,
			h.extractGVK,
			h.logger.WithGroup("statusUpdater"),
		),
		newTree.GatewayClasses,
		newTree.IgnoredGatewayClasses,
	)

	statusUpdater.UpdateStatus(ctx)

	// h.updateHAProxy(ctx, logger)  //revive:disable:unused-parameters
	// h.updateStatuses(ctx, logger) //revive:disable:unused-parameter
}
