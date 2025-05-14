//
// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	constant "github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/handler"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/predicate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	apiv1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8spredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Controller struct {
	Configuration config.Configuration
}

func New(options ...func(*config.Configuration) error) (Controller, error) {
	slogger := slog.Default()
	ctrl := Controller{
		Configuration: config.Configuration{
			Logger: slogger,
			K8sLogging: &config.K8sLogging{
				LogConverter: config.NewIOWriter(slogger),
			},
		},
	}
	for _, o := range options {
		err := o(&ctrl.Configuration)
		if err != nil {
			return Controller{}, err
		}
	}
	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	mgr, err := createManager(c.Configuration)
	if err != nil {
		return fmt.Errorf("cannot build runtime manager: %w", err)
	}

	eventCh := make(chan any)

	if err := registerControllers(ctx, c.Configuration, mgr, eventCh); err != nil {
		return fmt.Errorf("cannot register controllers: %w", err)
	}

	extractGVK := utils.NewExtractGKV(scheme, c.Configuration.Logger)

	treeBuilderConfig := handler.NewGateTreeBuilderConfig(
		mgr.GetClient(),
		mgr.GetAPIReader(),
		c.Configuration.GatewayClass,
		extractGVK,
	)

	eventHandler := handler.NewEventHandlerImpl(
		treeBuilderConfig,
		extractGVK,
		c.Configuration.Logger,
	)

	eventLoop := handler.NewEventLoop(
		eventCh,
		*c.Configuration.Logger,
		eventHandler,
	)

	if err = mgr.Add(eventLoop); err != nil {
		return fmt.Errorf("cannot register event loop: %w", err)
	}

	if err = mgr.Start(ctx); err != nil {
		return fmt.Errorf("cannot start runtime manager: %w", err)
	}

	return nil
}

func registerControllers( //revive:disable:function-length
	ctx context.Context,
	cfg config.Configuration,
	mgr manager.Manager,
	eventCh chan any,
) error {
	type ctlrCfg struct {
		name       string
		objectType client.Object
		options    []Option
	}

	crdWithGVK := apiext.CustomResourceDefinition{}
	crdWithGVK.SetGroupVersionKind(
		schema.GroupVersionKind{Group: apiext.GroupName, Version: "v1", Kind: "CustomResourceDefinition"},
	)

	controllerRegisterCfgs := []ctlrCfg{
		{
			// watch metadata of Gateway API CRDs
			// Gateway API CRDs are filtered with the predicate: AnnotationPredicate
			// on
			name:       "GatewayApiCRD",
			objectType: &crdWithGVK,
			options: []Option{
				WithOnlyMetadata(),
				WithK8sPredicate(
					k8spredicate.And(
						// k8spredicate.GenerationChangedPredicate{},
						predicate.AnnotationPredicate{Annotation: constant.BundleVersionAnnotation}),
				),
			},
		},
		{
			name:       "GatewayClass",
			objectType: &gatewayv1.GatewayClass{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.GatewayClassPredicate{ControllerName: cfg.GatewayCtlrName},
					),
				),
			},
		},
		{
			name:       "Gateway",
			objectType: &gatewayv1.Gateway{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.GatewayPredicate{GatewayClassName: cfg.GatewayClass},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
		{
			name:       "HTTPRoute",
			objectType: &gatewayv1.HTTPRoute{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.GatewayClassPredicate{ControllerName: cfg.GatewayCtlrName},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
		{
			name:       "Service",
			objectType: &apiv1.Service{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
		{
			name:       "Secret",
			objectType: &apiv1.Secret{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
		{
			name:       "EndpointSlice",
			objectType: &discoveryV1.EndpointSlice{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
				WithFieldIndices(index.CreateEndpointSliceFieldIndices()),
			},
		},
		{
			name:       "Namespace",
			objectType: &apiv1.Namespace{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
		{
			name:       "ConfigMap",
			objectType: &apiv1.ConfigMap{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
		{
			name:       "HaproxyGate",
			objectType: &v3.HaproxyGate{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.WhiteListNamespaces),
					),
				),
			},
		},
	}

	for _, registerConfig := range controllerRegisterCfgs {
		if err := Register(
			ctx,
			registerConfig.objectType,
			registerConfig.name,
			mgr,
			eventCh,
			registerConfig.options...,
		); err != nil {
			return fmt.Errorf("cannot register controller for %T: %w", registerConfig.objectType, err)
		}
	}
	return nil
}
