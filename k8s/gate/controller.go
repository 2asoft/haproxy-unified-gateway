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
	"os"
	"sync"

	"github.com/go-logr/logr"
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	hapi "github.com/haproxytech/kubernetes-controller/hug/haproxy/api"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	constant "github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/handler"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"
	objtypes "github.com/haproxytech/kubernetes-controller/k8s/gate/object-types"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/predicate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	apiv1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8spredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Controller struct {
	// HAProxyClient is set if Configuration.UpdateHaproxyThroughRuntime is true
	HaproxyClient hapi.HAProxyClient
	Configuration config.Configuration
}

func getControllerPodConfig() (config.ControllerPodConfig, error) {
	podIP, err := getValueFromEnv("POD_IP")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	ns, err := getValueFromEnv("POD_NAMESPACE")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	name, err := getValueFromEnv("POD_NAME")
	if err != nil {
		return config.ControllerPodConfig{}, err
	}

	c := config.ControllerPodConfig{
		PodIP:     podIP,
		Namespace: ns,
		Name:      name,
	}

	return c, nil
}

func getValueFromEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("environment variable %s not set", key)
	}

	return val, nil
}

func New(options config.GateConfigOptions) (Controller, error) {
	ctrl := Controller{Configuration: config.Configuration{}}
	for _, o := range options {
		err := o(&ctrl.Configuration)
		if err != nil {
			return Controller{}, err
		}
	}
	// --------------
	// Apply Defaults
	ctrl.Configuration.ApplyDefaults()

	// Logs inits
	logrLoggerFromSlog := logr.FromSlogHandler(ctrl.Configuration.LogHandler)
	runtimelog.SetLogger(logrLoggerFromSlog)

	// Other
	ctrlPodConfig, err := getControllerPodConfig()
	if err != nil {
		return Controller{}, err
	}
	ctrl.Configuration.ControllerPodConfig = ctrlPodConfig

	return ctrl, nil
}

func (c *Controller) Run(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	defer wg.Done()

	mgr, err := createManager(c.Configuration)
	if err != nil {
		return fmt.Errorf("cannot build runtime manager: %w", err)
	}

	if err := Add(ctx, c.Configuration, c.HaproxyClient, mgr); err != nil {
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("cannot start runtime manager: %w", err)
	}

	return nil
}

func Add(
	ctx context.Context,
	cfg config.Configuration,
	haproxyClient hapi.HAProxyClient,
	mgr manager.Manager,
) error {
	// Check if the controller configuration is valid
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid controller configuration: %w", err)
	}

	eventCh := make(chan any)

	if err := registerControllers(ctx, cfg, mgr, eventCh); err != nil {
		return fmt.Errorf("cannot register controllers: %w", err)
	}

	extractGVK := utils.NewExtractGKV(scheme, cfg.Logger)

	clusterStore := &store.ClusterStore{
		GatewayClasses:  make(map[types.NamespacedName]*gatewayv1.GatewayClass),
		Gateways:        make(map[types.NamespacedName]*gatewayv1.Gateway),
		HTTPRoutes:      make(map[types.NamespacedName]*gatewayv1.HTTPRoute),
		Services:        make(map[types.NamespacedName]*apiv1.Service),
		Namespaces:      make(map[types.NamespacedName]*apiv1.Namespace),
		Secrets:         make(map[types.NamespacedName]*apiv1.Secret),
		ConfigMaps:      make(map[types.NamespacedName]*apiv1.ConfigMap),
		GatewayAPICRDs:  make(map[types.NamespacedName]*metav1.PartialObjectMetadata),
		HugGates:        make(map[types.NamespacedName]*v3.HugGate),
		ControllerConfs: make(map[types.NamespacedName]*v3.HugConf),
		EndpointSlices:  make(map[types.NamespacedName]*discoveryV1.EndpointSlice),
		BackendCRs:      make(map[types.NamespacedName]*v3.Backend),
		Updates:         store.NewClusterUpdates(),
	}

	var certificateStorage storage.CertificateStorage
	var err error

	certificateStorage, err = storage.NewCertificateStorage(cfg.Logger, extractGVK, cfg.HaproxyParams.StoreCertificateStructureType,
		cfg.HaproxyParams.CertsDir, cfg.HaproxyParams.CertListDir)
	if err != nil {
		return err
	}

	mapsStorage, err := storage.NewMapsStorage(cfg.Logger, extractGVK, cfg.HaproxyParams.StoreMapsStructureType, cfg.HaproxyParams.MapsDir)
	if err != nil {
		return err
	}

	gateTreeConfig := handler.GateTreeConfig{
		BaseLogger:                 cfg.Logger,
		LogCategoryFilterHandler:   cfg.LogHandler,
		ExtractGVK:                 extractGVK,
		ControllerConfNsName:       cfg.HugConfCRD,
		TransferHaproxyConfChannel: cfg.TransferHaproxyConfChannel,
		K8sClient:                  mgr.GetClient(),
		K8sReader:                  mgr.GetAPIReader(),
		StoreCertificateOnDisk:     cfg.HaproxyParams.StoreCertificateOnDisk,
		StoreMapsOnDisk:            cfg.HaproxyParams.StoreMapsOnDisk,
		RuntimeUpdateHaproxy:       cfg.HaproxyParams.RuntimeUpdateHaproxy,
		CertificateStorage:         certificateStorage,
		MapsStorage:                mapsStorage,
		ControllerName:             cfg.ControllerName,
	}
	haproxyCfgMgrParams, err := haproxy.NewHaproxyConfMgrParams(extractGVK, cfg.HaproxyParams, certificateStorage, mapsStorage)
	if err != nil {
		return err
	}

	eventHandler := handler.NewEventHandlerImpl(
		clusterStore,
		gateTreeConfig,
		haproxyCfgMgrParams,
		cfg.InitialStructuredHaproxyConf,
		haproxyClient,
	)

	loopCfg := handler.EventLoopConfig{
		SyncPeriod:        cfg.SyncPeriod,
		StartupSyncPeriod: cfg.StartupSyncPeriod,
	}
	eventLoop := handler.NewEventLoop(
		loopCfg,
		eventCh,
		cfg.Logger,
		eventHandler,
		extractGVK,
	)

	if err := mgr.Add(eventLoop); err != nil {
		return fmt.Errorf("cannot register event loop: %w", err)
	}

	return nil
}

//revive:disable:function-length
func registerControllers(ctx context.Context, cfg config.Configuration, mgr manager.Manager, eventCh chan any) error {
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
			objectType: objtypes.ObjectTypeGatewayClass,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.GatewayClassPredicate{ControllerName: cfg.ControllerName},
					),
				),
			},
		},
		{
			name:       "Gateway",
			objectType: objtypes.ObjectTypeGateway,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
			},
		},
		{
			name:       "HTTPRoute",
			objectType: objtypes.ObjectTypeHTTPRoute,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						// predicate.GatewayPredicate{GatewayClassNames: cfg.GatewayClasses},
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
			},
		},
		{
			name:       "Service",
			objectType: objtypes.ObjectTypeService,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.GenerationChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
			},
		},
		{
			name:       "Secret",
			objectType: objtypes.ObjectTypeSecret,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
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
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
				WithFieldIndices(index.CreateEndpointSliceFieldIndices(cfg.Logger)),
			},
		},
		{
			name:       "Namespace",
			objectType: &apiv1.Namespace{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
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
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
			},
		},
		{
			name:       "HugGate",
			objectType: objtypes.ObjectTypeHugGate,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
			},
		},
		{
			name:       "ControllerConf",
			objectType: &v3.HugConf{},
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
						predicate.ControllerConfPredicate{
							ControllerConfName: cfg.HugConfCRD,
						},
					),
				),
			},
		},
		{
			name:       "BackendCR",
			objectType: objtypes.ObjectTypeBackend,
			options: []Option{
				WithK8sPredicate(
					k8spredicate.And(
						k8spredicate.ResourceVersionChangedPredicate{},
						predicate.NewNamespacePredicate(cfg.Namespaces),
					),
				),
			},
		},
	}

	for _, registerConfig := range controllerRegisterCfgs {
		params := registerParams{
			ctx:        ctx,
			logger:     cfg.Logger,
			objectType: registerConfig.objectType,
			name:       registerConfig.name,
			mgr:        mgr,
			eventCh:    eventCh,
			options:    registerConfig.options,
		}
		if err := Register(params); err != nil {
			return fmt.Errorf("cannot register controller for %T: %w", registerConfig.objectType, err)
		}
	}
	return nil
}
