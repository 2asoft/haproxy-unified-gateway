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

func (cfg *Configuration) ApplyDefaults() {
	// Logging Defaults
	slogger, logHandler := NewGateLogger(logging.DefaultLevel, logging.DefaultLogLevelPerCategory)
	cfg.Logger = slogger
	cfg.LogHandler = logHandler

	cfg.LeaderElectionConfig.LockName = "unified-controller-leader-election-lock"
	if cfg.ControllerName == "" {
		cfg.ControllerName = "gate.haproxy.org/unified-controller"
	}
	if cfg.SyncPeriod == 0 {
		cfg.SyncPeriod = 5 * time.Second
	}
}
