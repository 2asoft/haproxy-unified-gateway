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
package haproxy

import (
	"context"
	"log/slog"

	"github.com/haproxytech/client-native/v6/runtime"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/diffs"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/metadata"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/structured"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
)

type HaproxyConfMgr interface {
	// ComputeDiffs computes the HAProxy configuration diffs.
	ComputeDiffs() error
	GetDiffs() diffs.HaproxyConfDiffs
}

var _ HaproxyConfMgr = &HaproxyConfMgrImpl{}

type HaproxyConfMgrImpl struct {
	controllerStore tree.ControllerStore
	// frontendsOwnedbyGateway keeps track of frontends owned by each Gateway
	// This is usefull to cleanup the frontends removed from a Gateway (some listeners removed)
	frontendsOwnedbyGateway FrontendsOwnedbyGateway // map[gwKey] -> map[frontendName]struct{}
	metadataManager         metadata.Manager
	// RuntimeClient is set if HaproxyConfMgrParams.UpdateHaproxyThroughRuntime is true
	runtimeClient runtime.Runtime
	logger        *slog.Logger
	// frontendsContainedInFirstSync that are present at startup, used to cleanup after the first sync
	// the frontends that are not anymore in the cluster
	frontendsContainedInFirstSync map[string]struct{}
	configuration                 Configuration
	params                        HaproxyConfMgrParams
	// If this is the initial sync, we will add to the diffs Deleted all items that are not upserted
	firstSync bool // True if this is the initial sync
}

func NewHaproxyConfMgr(logger *slog.Logger, controllerStore tree.ControllerStore, startupStructured structured.Structured,
	params HaproxyConfMgrParams, runtimeClient runtime.Runtime,
) HaproxyConfMgr {
	firstSync := true
	impl := HaproxyConfMgrImpl{
		controllerStore: controllerStore,
		configuration: Configuration{
			structured: startupStructured,
		},
		firstSync:                     firstSync,
		params:                        params,
		logger:                        logger.With(logging.LogAttrCategory(logging.LogCategoryHaproxyCfgMgr)),
		frontendsContainedInFirstSync: make(map[string]struct{}),
		frontendsOwnedbyGateway:       NewFrontendsOwnedbyGateway(),
		metadataManager:               metadata.NewManager(params.extractGVK, params.LinkID),
		runtimeClient:                 runtimeClient,
	}

	return &impl
}

func (b *HaproxyConfMgrImpl) ComputeDiffs() error {
	logger := b.logger
	logger.LogAttrs(context.Background(), slog.LevelDebug, "Start computing HAProxy configuration diffs")
	defer logger.LogAttrs(context.Background(), slog.LevelDebug, "Finished computing HAProxy configuration diffs")

	// Clear the previous configuration diffs
	// This is important to ensure that we only transfer the current configuration changes.
	b.configuration.resetDiffs()

	// Build HAProxy configuration for the Gateways
	if err := b.processGateways(); err != nil {
		logger.LogAttrs(context.Background(), slog.LevelError, "Failed to build Gateways",
			logging.LogAttrError(err))
	}
	// Perform the needed cleanup after the first sync
	// Remove frontends that were present at startup but not anymore in the cluster
	b.cleanupAfterFirstSync()

	return nil
}

func (b *HaproxyConfMgrImpl) GetDiffs() diffs.HaproxyConfDiffs {
	return b.configuration.diffs
}

func (b *HaproxyConfMgrImpl) cleanupAfterFirstSync() {
	if !b.firstSync {
		return
	}

	// Check all frontends that are in the configuration but not any more in the cluster
	// If the initial sync period was too short, the impact is that we would send a Delete on the frontend
	// to the application.
	// But we would receive an Upsert on the next sync
	// The state will be eventually consistent
	for feName := range b.configuration.structured.Frontends {
		_, toKeep := b.frontendsContainedInFirstSync[feName]
		if !toKeep {
			b.logger.LogAttrs(context.Background(), slog.LevelDebug, "Frontend [DELETE STARTUP]",
				logging.LogAttrFrontendName(feName),
			)
			if err := b.configuration.deleteFrontend(b.logger, feName); err != nil {
				slog.LogAttrs(context.Background(), slog.LevelError, "Failed to delete frontend [startup]",
					logging.LogAttrFrontendName(feName))
				// continue to delete the rest of the frontends
			}
		}
	}

	b.frontendsContainedInFirstSync = make(map[string]struct{})
	b.firstSync = false
}
