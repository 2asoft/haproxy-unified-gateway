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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HaproxyCfgBuilder interface {
	// BuildHaproxyConf builds the HAProxy configuration.
	BuildHaproxyConf() error
}

var _ HaproxyCfgBuilder = &HaproxyConfBuilderImpl{}

type Templates struct {
	frontendNameTemplate string
	backendNameTemplate  string
	serverNameTemplate   string
}

type HaproxyConfBuilderParams struct {
	exctractGVK utils.ExtractGVK
	Templates
	iPV4BindAddr string
	iPV6BindAddr string
	linkID       string
	disableIPv4  bool
	disableIPv6  bool
}

type HaproxyConfBuilderImpl struct {
	controllerStore    tree.ControllerStore
	cfgDiffs           HaproxyCfgDiffs
	structuredCfgStore HaproxyCfg
	logger             *slog.Logger
	frontendsByGateway map[client.ObjectKey]map[string]struct{}
	params             HaproxyConfBuilderParams
}

func NewTemplates(feTemplate, beTemplate, seTemplate string) Templates {
	return Templates{
		frontendNameTemplate: feTemplate,
		backendNameTemplate:  beTemplate,
		serverNameTemplate:   seTemplate,
	}
}

func NewHaproxyCfgBuilderParams(extractGVK utils.ExtractGVK,
	templates Templates,
	disableIPv4, disableIPv6 bool,
	iPV4BindAddr, iPV6BindAddr string,
	linkID string,
) HaproxyConfBuilderParams {
	return HaproxyConfBuilderParams{
		exctractGVK:  extractGVK,
		Templates:    templates,
		disableIPv4:  disableIPv4,
		disableIPv6:  disableIPv6,
		iPV4BindAddr: iPV4BindAddr,
		iPV6BindAddr: iPV6BindAddr,
		linkID:       linkID,
	}
}

func NewHaproxyConfBuilder(controllerStore tree.ControllerStore, haproxyCfgStore HaproxyCfg, builderConfig HaproxyConfBuilderParams) HaproxyConfBuilderImpl {
	return HaproxyConfBuilderImpl{
		controllerStore:    controllerStore,
		structuredCfgStore: haproxyCfgStore,
		params:             builderConfig,
		logger:             controllerStore.Logger,
		cfgDiffs:           HaproxyCfgDiffs{Created: NewHaproxyCfg(), Updated: NewHaproxyCfg(), Deleted: NewHaproxyCfg()},
		frontendsByGateway: make(map[client.ObjectKey]map[string]struct{}),
	}
}

func (b *HaproxyConfBuilderImpl) BuildHaproxyConf() error {
	logger := b.logger
	logger.LogAttrs(context.Background(), slog.LevelDebug,
		"Start building HAProxy configuration",
	)

	// Clear the previous configuration diffs
	// This is important to ensure that we only transfer the current configuration changes.
	b.cfgDiffs = HaproxyCfgDiffs{
		Created: NewHaproxyCfg(),
		Updated: NewHaproxyCfg(),
		Deleted: NewHaproxyCfg(),
	}

	// Build HAProxy configuration for the frontends
	if err := b.buildFrontends(); err != nil {
		logger.LogAttrs(context.Background(), slog.LevelError,
			"Failed to build frontends",
			logging.LogAttrError(err))
	}

	return nil
}

func (b *HaproxyConfBuilderImpl) GetCfsDiffs() HaproxyCfgDiffs {
	return b.cfgDiffs
}
