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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/events"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/status"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	discoveryV1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventHandler handles a batch of events.
// Its builds a ClusterStore of k8s resources.
type EventHandler interface {
	// HandleEventBatch handles a batch of events.
	// EventBatch can include duplicated events.
	HandleEventBatch(ctx context.Context, batch events.EventBatch)
}

type GateTreeConfig struct {
	// k8sClient is a Kubernetes API client.
	K8sClient client.Client
	// k8sReader is a Kubernets API reader.
	K8sReader                  client.Reader
	BaseLogger                 *slog.Logger
	LogCategoryFilterHandler   *logging.CategoryFilterHandler
	ExtractGVK                 utils.ExtractGVK
	TransferHaproxyConfChannel chan haproxy.HaproxyCfgDiffs
	//  Namespace and name of the controller conf CRD
	ControllerConfNsName types.NamespacedName
}

// eventHandlerImpl implements EventHandler.
// eventHandlerImpl is responsible for:
// - Reconciling the Gateway API and Kubernetes built-in resources with the HAProxy configuration.
// - building the GateTree
type eventHandlerImpl struct {
	haproxyConfBuilder  haproxy.HaproxyConfMgrImpl
	config              GateTreeConfig
	clusterStoreUpdater store.ClusterStoreUpdater
	treeBuilder         GateTreeBuilder
}

// NewEventHandlerImpl creates a new eventHandlerImpl.
func NewEventHandlerImpl(
	clusterStore *store.ClusterStore,
	gateTreeConfig GateTreeConfig,
	haproxyCfgBuilderConfig haproxy.HaproxyConfMgrParams,
) *eventHandlerImpl {
	clusterStoreUpdater := store.NewClusterStoreUpdaterImpl(
		clusterStore,
		gateTreeConfig.ExtractGVK,
		gateTreeConfig.BaseLogger,
	)

	gateTree := tree.NewGateTree()
	unmanagedGateTree := tree.NewGateTree()
	referencedObjects := tree.NewReferencedObjects(gateTreeConfig.ExtractGVK)

	controllerStore := tree.ControllerStore{
		ClusterStore:      clusterStore,
		GateTree:          gateTree,
		ReferencedObjects: referencedObjects,
		UnmanagedGateTree: unmanagedGateTree,
		ExtractGVK:        gateTreeConfig.ExtractGVK,
		Logger:            gateTreeConfig.BaseLogger.With(logging.LogAttrCategory(logging.LogCategoryGate)),
		InstalledGwAPIVersions: &tree.InstalledVersions{
			Versions: make(map[string]int),
		},
	}

	treeBuilder := NewGateTreeBuilder(
		controllerStore,
		gateTreeConfig,
	)

	haproxyCfgStore := haproxy.NewHaproxyCfg()
	haproxyConfBuilder := haproxy.NewHaproxyConfBuilder(gateTreeConfig.BaseLogger, controllerStore, haproxyCfgStore, haproxyCfgBuilderConfig)

	handler := &eventHandlerImpl{
		treeBuilder:         treeBuilder,
		config:              gateTreeConfig,
		clusterStoreUpdater: clusterStoreUpdater,
		haproxyConfBuilder:  haproxyConfBuilder,
	}

	return handler
}

func (h *eventHandlerImpl) HandleEventBatch(ctx context.Context, batch events.EventBatch) {
	start := time.Now()

	h.config.BaseLogger.LogAttrs(context.Background(), slog.LevelInfo,
		"Started processing event batch",
		logging.LogAttrBatch(batch.BatchID, len(batch.Events)),
	)

	defer func() {
		duration := time.Since(start)
		h.config.BaseLogger.LogAttrs(context.Background(), slog.LevelInfo,
			"Finished processing event batch",
			logging.LogAttrBatch(batch.BatchID, len(batch.Events)),
			logging.LogAttrDuration(duration),
		)
	}()

	// Process each event in the batch
	_ = h.processBatch(batch)

	// Build the GateTree
	h.treeBuilder.buildGateTree()
	gatetree := h.treeBuilder.GetTree()

	// HAProxy Configuration building
	err := h.haproxyConfBuilder.UpdateHaproxyConf()
	if err != nil {
		h.config.BaseLogger.LogAttrs(context.Background(), slog.LevelError,
			"error building HAProxy configuration",
			logging.LogAttrError(err),
		)
	}
	haproxyConfDiffs := h.haproxyConfBuilder.GetCfsDiffs()
	if !haproxyConfDiffs.IsEmpty() {
		if h.config.TransferHaproxyConfChannel != nil {
			h.config.TransferHaproxyConfChannel <- haproxyConfDiffs
		}
	}

	// START EXAMPLE
	// Below is just an example
	// We list EndpointSlices using the Service Name Index Field we added as an index to the EndpointSlice cache.
	// This allows us to perform a quick lookup of all EndpointSlices for a Service.
	var endpointSliceList discoveryV1.EndpointSliceList
	svcName := "http-echo"
	svcNs := "default"
	err = h.treeBuilder.cfg.K8sClient.List(
		ctx,
		&endpointSliceList,
		client.MatchingFields{index.EndpointSliceServiceNameIndexField: svcName},
		client.InNamespace(svcNs),
	)
	if err != nil {
		h.config.BaseLogger.LogAttrs(context.Background(), slog.LevelError,
			"could not retrieve http-echo endpoints",
			logging.LogAttrError(err),
		)
	}
	// h.config.Logger.Info(
	// 	fmt.Sprintf("JUST AN EXAMPLE to show cache indexes usage. eps for http-echo svc %v", endpointSliceList))
	// END EXAMPLE

	statusUpdater := status.NewStatusUpdaterImpl(
		status.NewStatusUpdaterConf(
			h.treeBuilder.cfg.K8sClient,
			h.config.ExtractGVK,
			h.config.BaseLogger,
		),
		gatetree.GatewayClasses,
		gatetree.Gateways,
	)

	statusUpdater.UpdateStatus(ctx)

	// h.treeBuilder.cleanGateTreeUpdates()
	// h.updateHAProxy(ctx, logger)  //revive:disable:unused-parameters
	// h.updateStatuses(ctx, logger) //revive:disable:unused-parameter
}

func (h *eventHandlerImpl) processBatch(batch events.EventBatch) bool {
	h.clusterStoreUpdater.ResetUpdates()

	for _, e := range batch.Events {
		h.updateClusterStore(e, h.config.BaseLogger)
	}
	return true
}

func (h *eventHandlerImpl) updateClusterStore(event any, logger *slog.Logger) {
	switch obj := event.(type) {
	case *events.UpsertEvent:
		gvk := h.config.ExtractGVK(obj.Resource)
		logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrEventType("upsert"),
			logging.LogAttrResource(obj.Resource, gvk),
		)

		h.clusterStoreUpdater.Upsert(obj.Resource)

	case *events.DeleteEvent:
		gvk := h.config.ExtractGVK(obj.Type)

		logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrEventType("delete"),
			logging.LogAttrResource(obj.Type, gvk),
		)

		h.clusterStoreUpdater.Delete(obj.Type, obj.NamespacedName)
	}
}
