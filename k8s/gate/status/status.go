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
package status

import (
	"context"
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusUpdater interface {
	UpdateStatus(ctx context.Context)
}

type StatusUpdatedConf struct {
	logger     *slog.Logger
	client     client.Client
	extractGVK utils.ExtractGVK
}

type StatusUpdaterImpl struct {
	cfg              StatusUpdatedConf
	GatewayClasses   map[types.NamespacedName]*tree.GatewayClass
	ignoredGwClasses map[types.NamespacedName]*tree.GatewayClass
}

func NewStatusUpdaterImpl(
	cfg StatusUpdatedConf,
	gatewayClasses map[types.NamespacedName]*tree.GatewayClass,
	ignoredGwClasses map[types.NamespacedName]*tree.GatewayClass,
) *StatusUpdaterImpl {
	return &StatusUpdaterImpl{
		cfg:              cfg,
		GatewayClasses:   gatewayClasses,
		ignoredGwClasses: ignoredGwClasses,
	}
}

func NewStatusUpdaterConf(
	client client.Client,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) StatusUpdatedConf {
	return StatusUpdatedConf{
		logger:     logger,
		extractGVK: extractGVK,
		client:     client,
	}
}

var _ StatusUpdater = &StatusUpdaterImpl{}

func (s *StatusUpdaterImpl) UpdateStatus(ctx context.Context) {
	// this is just a beginning, needs to be better design
	// let's start with something very basic that updates only GatewayClass status for now

	for _, gwc := range s.GatewayClasses {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.cfg.logger.Info(
			"Updating status for resource",
			"namespace", gwc.K8sResource.Namespace,
			"name", gwc.K8sResource.Name,
			"kind", s.cfg.extractGVK(gwc.K8sResource),
		)

		s.writeGatewayClassStatus(ctx, gwc)
	}
}
