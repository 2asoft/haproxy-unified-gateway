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
package reload

import (
	"context"
	"log/slog"
	"sync"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
)

// The singleton instance
var (
	rlm  *reloadMgr
	once sync.Once
)

// The singleton type
type reloadMgr struct {
	logger *slog.Logger
	reload bool
}

// GetInstance returns the singleton instance.
func GetInstance() *reloadMgr {
	once.Do(func() {
		rlm = &reloadMgr{} // Initialization logic
	})
	return rlm
}

func (rlm *reloadMgr) SetLogger(logger *slog.Logger) {
	rlm.logger = logger.With(logging.LogAttrCategory(logging.LogCategoryReloadMgr))
}

func (rlm *reloadMgr) SetReload(reason string, args ...any) {
	rlm.reload = true
	if !rlm.validReason(reason) {
		return
	}
	if rlm.logger != nil {
		rlm.logger.LogAttrs(context.WithValue(context.Background(), logging.CallerAdditionalSkipKey, 1), slog.LevelInfo,
			"reload required",
			logging.LogAttrReloadMgrAction(rlm.reload, reason, args...))
	}
}

func (rlm *reloadMgr) Reset() {
	if rlm.reload && rlm.logger != nil {
		rlm.logger.LogAttrs(context.WithValue(context.Background(), logging.CallerAdditionalSkipKey, 1), slog.LevelInfo,
			"reload reset",
		)
	}
	rlm.reload = false
}

func (rlm *reloadMgr) NeedReload() bool {
	return rlm.reload
}

func (*reloadMgr) validReason(reason string) bool {
	return reason != ""
}
