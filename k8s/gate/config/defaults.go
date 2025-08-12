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
package config

import (
	"time"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
)

const (
	// defaultControllerName is the default name of the controller.
	defaultControllerName         = "gate.haproxy.org/unified-controller"
	defaultLeaderElectionLockName = "unified-controller-leader-election-lock"
	defaultSyncPeriod             = 5 * time.Second
	defaultFrontendNameTemplate   = "{{ .LINK_ID }}_{{ .GATEWAY_NAMESPACE}}_{{ .GATEWAY_NAME }}_{{ .LISTENER_NAME }}"
	defaultBackendNameTemplate    = ""
	defaultServerNameTemplate     = ""
	DefaultsSectionName           = "haproxytech"
)

func (cfg *Configuration) ApplyDefaults() {
	if cfg.LogHandlerType == "" {
		cfg.LogHandlerType = logging.LogHandlerTypeJSON
	}
	// Logging Defaults
	slogger, logHandler := NewBaseLogger(cfg.LogHandlerType, logging.DefaultLevel, logging.DefaultLogLevelPerCategory)
	cfg.Logger = slogger
	cfg.LogHandler = logHandler

	cfg.LeaderElectionConfig.LockName = defaultLeaderElectionLockName
	if cfg.ControllerName == "" {
		cfg.ControllerName = defaultControllerName
	}
	if cfg.SyncPeriod == 0 {
		cfg.SyncPeriod = defaultSyncPeriod
	}
	// StartupSyncPeriod if not defined is equal to SyncPeriod
	if cfg.StartupSyncPeriod == 0 {
		cfg.StartupSyncPeriod = cfg.SyncPeriod
	}
	if cfg.FrontendNameTemplate == "" {
		cfg.FrontendNameTemplate = defaultFrontendNameTemplate
	}
	if cfg.BackendNameTemplate == "" {
		cfg.BackendNameTemplate = defaultBackendNameTemplate
	}
	if cfg.ServerNameTemplate == "" {
		cfg.ServerNameTemplate = defaultServerNameTemplate
	}
	if cfg.LinkID == "" {
		cfg.LinkID = "linkid"
	}
}
