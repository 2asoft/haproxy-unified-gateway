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
	"context"
	"log/slog"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"k8s.io/apimachinery/pkg/types"
)

type HugConfBuilderImpl struct {
	ControllerStore
	logCategoryFilterHandler *logging.CategoryFilterHandler
	hugConfNsName            types.NamespacedName
}

type HugConfBuilderParams struct {
	ControllerStore
	LogCategoryFilterHandler *logging.CategoryFilterHandler
	HugConfNsName            types.NamespacedName
}

func NewHugConfBuilder(params HugConfBuilderParams) *HugConfBuilderImpl {
	return &HugConfBuilderImpl{
		ControllerStore:          params.ControllerStore,
		logCategoryFilterHandler: params.LogCategoryFilterHandler,
		hugConfNsName:            params.HugConfNsName,
	}
}

func (b *HugConfBuilderImpl) Build() {
	hugConfUpdates := b.ClusterStore.Updates.HugConfs
	if len(hugConfUpdates) == 0 {
		// no controller confs, nothing to do
		return
	}

	confUpdate, ok := hugConfUpdates[b.hugConfNsName]
	if !ok {
		// Update for controller conf not found, nothing to do
		return
	}

	// Reconcile the controller configuration
	switch confUpdate.Status {
	case store.StatusDeleted:
		// Case DELETED
		if confUpdate.Status == store.StatusDeleted {
			b.Logger.LogAttrs(context.Background(), slog.LevelInfo,
				"Resetting controller log configuration to defaults",
			)
			// Reset the log category filter handler to defaults
			b.logCategoryFilterHandler.ResetToDefaults()
			l, m := logging.GetLogSettings()
			b.Logger.LogAttrs(context.Background(), slog.LevelInfo,
				"Reconciled controller log configuration",
				logging.LogAttrLogSettings(l, m),
			)
			return
		}
	case store.StatusUpserted:
		// Case UPSERTED
		newConf := b.ClusterStore.ControllerConfs[b.hugConfNsName]
		if newConf == nil {
			b.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Controller configuration not found",
				logging.LogAttrNsName(b.hugConfNsName),
			)
			return
		}

		expectedLogCategoryPerLevel := make(map[v3.Category]slog.Level)
		for _, catLevel := range newConf.Spec.Logging.CategoryLevelList {
			expectedLogCategoryPerLevel[catLevel.Category] = logging.LogLevelString2SlogLevel(string(catLevel.Level))
		}
		expectedLevel := logging.LogLevelString2SlogLevel(string(newConf.Spec.Logging.DefaultLevel))

		changed := b.logCategoryFilterHandler.ReconcileLogSettings(expectedLevel, expectedLogCategoryPerLevel)
		if changed {
			l, m := logging.GetLogSettings()
			b.Logger.LogAttrs(context.Background(), slog.LevelInfo,
				"Reconciled controller log configuration",
				logging.LogAttrLogSettings(l, m),
			)
		}
	}
}

func (*HugConfBuilderImpl) BuildStatus() {
}
