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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusUpdater interface {
	UpdateStatus(ctx context.Context)
}

type StatusUpdaterConf struct {
	logger     *slog.Logger
	client     client.Client
	extractGVK utils.ExtractGVK
}

type StatusUpdaterImpl struct {
	config         StatusUpdaterConf
	GatewayClasses map[types.NamespacedName]*tree.GatewayClass
	Gateways       map[types.NamespacedName]*tree.Gateway
}

func NewStatusUpdater(
	cfg StatusUpdaterConf,
	gatewayClasses map[types.NamespacedName]*tree.GatewayClass,
	gateways map[types.NamespacedName]*tree.Gateway,
) StatusUpdater {
	return &StatusUpdaterImpl{
		config:         cfg,
		GatewayClasses: gatewayClasses,
		Gateways:       gateways,
	}
}

func NewStatusUpdaterConf(
	k8sClient client.Client,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) StatusUpdaterConf {
	return StatusUpdaterConf{
		logger:     logger.With(logging.LogAttrCategory(logging.LogCategoryStatus)),
		extractGVK: extractGVK,
		client:     k8sClient,
	}
}

var _ StatusUpdater = &StatusUpdaterImpl{}

func (s *StatusUpdaterImpl) UpdateStatus(ctx context.Context) {
	// this is just a beginning, needs to be better design
	// let's start with something very basic that updates only GatewayClass status for now

	// GatewayClasses
	for _, gwc := range s.GatewayClasses {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Do not set Status for Deleted or unchanged GatewayClasses
		if gwc.TreeStatus.Status == store.StatusDeleted || gwc.TreeStatus.Status == "" {
			continue
		}
		// Do not set Status for Not managed GatewayClasses
		if !gwc.Managed {
			continue
		}

		s.config.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Updating status for resource",
			logging.LogAttrResource(gwc.K8sResource, s.config.extractGVK(gwc.K8sResource)),
		)

		s.writeGatewayClassStatus(ctx, gwc)
	}

	// Gateways
	for _, gw := range s.Gateways {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Do not set Status for Deleted or unchanged Gateways
		if gw.TreeStatus.Status == store.StatusDeleted || gw.TreeStatus.Status == "" {
			continue
		}

		s.config.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Updating status for resource",
			logging.LogAttrResource(gw.K8sResource, s.config.extractGVK(gw.K8sResource)),
		)

		s.writeGatewayStatus(ctx, gw)
	}

	// HTTPRoutes
	// TODO
	// END HTTPRoutes
}

type StatusUpdateParams[T client.Object] struct {
	Object           T
	StatusPatcher    StatusPatcher
	Getter           client.Client
	StatusUpdater    client.SubResourceWriter
	ConditionHandler generic.ConditionAccessor[T]
	Logger           *slog.Logger
	extractGVK       utils.ExtractGVK
	NsName           types.NamespacedName
}

func TryUpdateStatusFunc[T client.Object](param StatusUpdateParams[T]) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		objAttr := logging.LogAttrKeyGVK(param.NsName, param.extractGVK(param.Object))

		// Create a fresh empty object of type T
		clusterObj, ok := param.Object.DeepCopyObject().(T)
		if !ok {
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when copying object",
				objAttr)
			return false, nil
		}
		err := param.Getter.Get(ctx, types.NamespacedName{
			Namespace: param.NsName.Namespace,
			Name:      param.NsName.Name,
		}, clusterObj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when getting resource to update status",
				objAttr)
			return false, nil
		}

		statusAlreadyUpToDate, err := param.StatusPatcher.StatusEqual(clusterObj)
		if err != nil {
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when checking status equality",
				objAttr)
			return false, nil
		}

		if statusAlreadyUpToDate {
			param.Logger.LogAttrs(context.Background(), slog.LevelDebug,
				"Status already up to date",
				objAttr)
			return true, nil
		}

		// Status update
		if err := param.StatusPatcher.SetStatus(clusterObj); err != nil {
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when setting status",
				objAttr)
			return false, nil
		}
		if err := param.StatusUpdater.Update(ctx, clusterObj); err != nil {
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when updating status",
				objAttr,
				logging.LogAttrError(err))
			return false, nil
		}

		param.Logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Successfully updated status",
			objAttr,
		)
		return true, nil
	}
}
