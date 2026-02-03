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

	hapi "github.com/haproxytech/haproxy-unified-gateway/hug/haproxy/api"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/events"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/certificate"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/diffs"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/storage"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/structured"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/status"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/tree"
	utilsk8s "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils-k8s"

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
	CertificateStorage         storage.CertificateStorage
	MapsStorage                storage.MapsStorageEx
	BaseLogger                 *slog.Logger
	LogCategoryFilterHandler   *logging.CategoryFilterHandler
	ExtractGVK                 utilsk8s.ExtractGVK
	TransferHaproxyConfChannel chan diffs.HaproxyConfDiffs
	//  Namespace and name of the controller conf CRD
	ControllerConfNsName types.NamespacedName
	// ControllerName
	ControllerName string
	// StoreCertificatesOnDisk is a flag that indicates to the gate library to store certificates on disk
	StoreCertificateOnDisk bool
	// StoreMapsOnDisk is a flag that indicates to the gate library to store maps on disk
	StoreMapsOnDisk bool
	// RuntimeUpdateHaproxy
	RuntimeUpdateHaproxy bool
}

// eventHandlerImpl implements EventHandler.
// eventHandlerImpl is responsible for:
// - Reconciling the Gateway API and Kubernetes built-in resources with the HAProxy configuration.
// - building the GateTree
type eventHandlerImpl struct {
	clusterStoreUpdater store.ClusterStoreUpdater
	haproxyConfBuilder  haproxy.HaproxyConfMgr
	logger              *slog.Logger
	config              GateTreeConfig
	treeBuilder         GateTreeBuilder
}

// NewEventHandlerImpl creates a new eventHandlerImpl.
func NewEventHandlerImpl(
	clusterStore *store.ClusterStore,
	gateTreeConfig GateTreeConfig,
	haproxyCfgManagerParams haproxy.HaproxyConfMgrParams,
	initialStructuredConf structured.Structured,
	haproxyClient hapi.HAProxyClient,
) EventHandler {
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
		CertUpdates: &tree.CertUpdates{
			Created: make(map[string]certificate.CertificateData),
			Updated: make(map[string]certificate.CertificateData),
			Deleted: make(map[string]certificate.CertificateData),
		},
		CrtListUpdates: &tree.CrtListUpdates{
			Created: make(map[string]certificate.CrtListData),
			Updated: make(map[string]certificate.CrtListData),
			Deleted: make(map[string]certificate.CrtListData),
		},
		ControllerName: gateTreeConfig.ControllerName,
	}

	treeBuilder := NewGateTreeBuilder(&controllerStore, gateTreeConfig)

	haproxyConfMgr := haproxy.NewHaproxyConfMgr(gateTreeConfig.BaseLogger, &controllerStore, initialStructuredConf,
		haproxyCfgManagerParams, haproxyClient, gateTreeConfig.K8sClient)

	handler := &eventHandlerImpl{
		treeBuilder:         treeBuilder,
		config:              gateTreeConfig,
		clusterStoreUpdater: clusterStoreUpdater,
		haproxyConfBuilder:  haproxyConfMgr,
		logger:              gateTreeConfig.BaseLogger.With(logging.LogAttrCategory(logging.LogCategoryBatch)),
	}

	return handler
}

func (h *eventHandlerImpl) HandleEventBatch(ctx context.Context, batch events.EventBatch) {
	start := time.Now()

	h.logger.LogAttrs(context.Background(), slog.LevelInfo,
		"Started processing event batch",
		logging.LogAttrBatch(batch.BatchID, len(batch.Events)),
	)

	defer func() {
		duration := time.Since(start)
		h.logger.LogAttrs(context.Background(), slog.LevelInfo,
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
	err := h.haproxyConfBuilder.ComputeDiffs(ctx)
	if err != nil {
		h.logger.LogAttrs(context.Background(), slog.LevelError,
			"error building HAProxy configuration",
			logging.LogAttrError(err),
		)
	}
	haproxyConfDiffs := h.haproxyConfBuilder.GetDiffs()
	if !haproxyConfDiffs.IsEmpty() || haproxyConfDiffs.ReloadNeed {
		if h.config.TransferHaproxyConfChannel != nil {
			haproxyConfDiffs.Done = make(chan struct{})
			h.config.TransferHaproxyConfChannel <- haproxyConfDiffs
			<-haproxyConfDiffs.Done
		}
	}

	statusUpdater := status.NewStatusUpdater(
		status.NewStatusUpdaterConf(
			h.treeBuilder.cfg.K8sClient,
			h.config.ExtractGVK,
			h.config.ControllerName,
			h.config.BaseLogger,
		),
		gatetree.GatewayClasses,
		gatetree.Gateways,
		gatetree.HTTPRoutes,
		gatetree.TLSRoutes,
	)

	statusUpdater.UpdateStatus(ctx)
}

func (h *eventHandlerImpl) processBatch(batch events.EventBatch) bool {
	h.clusterStoreUpdater.ResetUpdates()

	for _, e := range batch.Events {
		h.updateClusterStore(e)
	}
	return true
}

func (h *eventHandlerImpl) updateClusterStore(event any) {
	switch obj := event.(type) {
	case *events.UpsertEvent:
		gvk := h.config.ExtractGVK(obj.Resource)
		h.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrEventType("upsert"),
			logging.LogAttrResource(obj.Resource, gvk),
		)

		h.clusterStoreUpdater.Upsert(obj.Resource)

	case *events.DeleteEvent:
		gvk := h.config.ExtractGVK(obj.Type)

		h.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Processing event in batch",
			logging.LogAttrEventType("delete"),
			logging.LogAttrResource(obj.Type, gvk),
		)

		h.clusterStoreUpdater.Delete(obj.Type, obj.NamespacedName)
	}
}
