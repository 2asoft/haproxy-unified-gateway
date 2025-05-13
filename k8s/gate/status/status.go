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
	client client.Client,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) StatusUpdaterConf {
	return StatusUpdaterConf{
		logger:     logger,
		extractGVK: extractGVK,
		client:     client,
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

		s.cfg.logger.Info(
			"Updating status for resource",
			"namespace", gwc.K8sResource.Namespace,
			"name", gwc.K8sResource.Name,
			"kind", s.cfg.extractGVK(gwc.K8sResource),
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
	ConditionHandler  conditions.ConditionHandler[T]
	extractGVK        utils.ExtractGVK
}

func TryUpdateStatusFunc[T client.Object](param StatusUpdateParams[T]) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		// Create a fresh empty object of type T
		obj := param.Object.DeepCopyObject().(T)
		err := param.Getter.Get(ctx, types.NamespacedName{
			Namespace: param.Object.GetNamespace(),
			Name:      param.Object.GetName(),
		}, obj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			param.Logger.Info(
				"Encountered error when getting resource to update status",
				"error", err,
				"object", client.ObjectKeyFromObject(obj),
				"kind", param.extractGVK(obj),
			)
			return false, nil
		}

		currentConditions := param.ConditionHandler.GetConditions(obj)
		if currentConditions.Equal(param.DesiredConditions) {
			param.Logger.Info(
				"Status already up to date",
				"object", client.ObjectKeyFromObject(obj),
				"kind", param.extractGVK(obj),
			)
			return true, nil
		}

		param.ConditionHandler.SetConditions(obj, param.DesiredConditions)

		if err := param.StatusUpdater.Update(ctx, obj); err != nil {
			param.Logger.Info(
				"Encountered error updating status",
				"error", err,
				"object", client.ObjectKeyFromObject(obj),
				"kind", param.extractGVK(obj),
			)
			return false, nil
		}

		param.Logger.Info(
			"Successfully updated status",
			"object", client.ObjectKeyFromObject(obj),
			"kind", param.extractGVK(obj),
		)
		return true, nil
	}
}
