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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
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
	cfg              StatusUpdaterConf
	GatewayClasses   map[types.NamespacedName]*tree.GatewayClass
	ignoredGwClasses map[types.NamespacedName]*tree.GatewayClass
}

func NewStatusUpdaterImpl(
	cfg StatusUpdaterConf,
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
	k8sClient client.Client,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) StatusUpdaterConf {
	return StatusUpdaterConf{
		logger:     logger,
		extractGVK: extractGVK,
		client:     k8sClient,
	}
}

var _ StatusUpdater = &StatusUpdaterImpl{}

func (s *StatusUpdaterImpl) UpdateStatus(ctx context.Context) {
	// this is just a beginning, needs to be better design
	// let's start with something very basic that updates only GatewayClass status for now

	// GatewayClasses
	gwcToUpdate := make(map[types.NamespacedName]*tree.GatewayClass)
	for k, gwc := range s.ignoredGwClasses {
		gwcToUpdate[k] = gwc
	}
	for k, gwc := range s.GatewayClasses {
		gwcToUpdate[k] = gwc
	}

	for _, gwc := range gwcToUpdate {
		select {
		case <-ctx.Done():
			return
		default:
		}

		s.cfg.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Updating status for resource",
			logging.LogAttrCategory(logging.LogCategoryStatus),
			logging.LogAttrResource(gwc.K8sResource, s.cfg.extractGVK(gwc.K8sResource)),
		)

		s.writeGatewayClassStatus(ctx, gwc)
	}
}

type StatusUpdateParams[T client.Object] struct {
	Object            T
	DesiredConditions conditions.Conditions
	Getter            client.Client
	StatusUpdater     client.SubResourceWriter
	Logger            *slog.Logger
	ConditionHandler  conditions.ConditionAccessor[T]
	extractGVK        utils.ExtractGVK
}

func TryUpdateStatusFunc[T client.Object](param StatusUpdateParams[T]) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		objAttr := logging.LogAttrResource(param.Object, param.extractGVK(param.Object))

		// Create a fresh empty object of type T
		obj, ok := param.Object.DeepCopyObject().(T)
		if !ok {
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when copying object",
				logging.LogAttrCategory(logging.LogCategoryStatus),
				objAttr)
			return false, nil
		}
		err := param.Getter.Get(ctx, types.NamespacedName{
			Namespace: param.Object.GetNamespace(),
			Name:      param.Object.GetName(),
		}, obj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when getting resource to update status",
				logging.LogAttrCategory(logging.LogCategoryStatus),
				objAttr)
			return false, nil
		}

		currentConditions := param.ConditionHandler.GetConditions(obj)
		if currentConditions.Equal(param.DesiredConditions) {
			param.Logger.LogAttrs(context.Background(), slog.LevelDebug,
				"Status already up to date",
				logging.LogAttrCategory(logging.LogCategoryStatus),
				objAttr)
			return true, nil
		}

		param.ConditionHandler.SetConditions(obj, param.DesiredConditions)

		if err := param.StatusUpdater.Update(ctx, obj); err != nil {
			param.Logger.LogAttrs(context.Background(), slog.LevelError,
				"Encountered error when updating status",
				logging.LogAttrCategory(logging.LogCategoryStatus),
				objAttr,
				logging.LogAttrError(err))
			return false, nil
		}

		param.Logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Successfully updated status",
			logging.LogAttrCategory(logging.LogCategoryStatus),
			objAttr,
		)
		return true, nil
	}
}
