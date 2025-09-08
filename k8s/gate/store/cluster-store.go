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
package store

import (
	"log/slog"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	v1 "k8s.io/api/core/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ClusterStore includes cluster resources necessary to build the Tree.
type ClusterStore struct {
	GatewayClasses  map[types.NamespacedName]*gatewayv1.GatewayClass
	Gateways        map[types.NamespacedName]*gatewayv1.Gateway
	HTTPRoutes      map[types.NamespacedName]*gatewayv1.HTTPRoute
	Services        map[types.NamespacedName]*v1.Service
	Namespaces      map[types.NamespacedName]*v1.Namespace
	Secrets         map[types.NamespacedName]*v1.Secret
	ConfigMaps      map[types.NamespacedName]*v1.ConfigMap
	GatewayAPICRDs  map[types.NamespacedName]*metav1.PartialObjectMetadata
	HaproxyGates    map[types.NamespacedName]*v3.HaproxyGate
	ControllerConfs map[types.NamespacedName]*v3.HugConf
	Updates         ClusterUpdates
}

// ClusterStoreUpdater updates the cluster store.
type ClusterStoreUpdater interface {
	Upsert(obj client.Object)
	Delete(obj client.Object, nsname types.NamespacedName)
	ResetUpdates()
}

type ClusterStoreUpdaterImpl struct {
	clusterStore *ClusterStore
	storeAdapter *storeAdapter
	extractGVK   utils.ExtractGVK
	logger       *slog.Logger
}

// to ensure that objectStoreImpl implements ObjectSore interface
var _ ClusterStoreUpdater = &ClusterStoreUpdaterImpl{}

func NewClusterStoreUpdaterImpl(
	clusterStore *ClusterStore,
	extractGVK utils.ExtractGVK,
	logger *slog.Logger,
) ClusterStoreUpdater {
	return &ClusterStoreUpdaterImpl{
		clusterStore: clusterStore,
		storeAdapter: &storeAdapter{
			stores: map[schema.GroupVersionKind]ObjectStoreUpdater{
				extractGVK(&gatewayv1.GatewayClass{}):          newObjectStoreImpl(clusterStore.GatewayClasses, clusterStore.Updates.GatewayClasses, logger),
				extractGVK(&gatewayv1.Gateway{}):               newObjectStoreImpl(clusterStore.Gateways, clusterStore.Updates.Gateways, logger),
				extractGVK(&gatewayv1.HTTPRoute{}):             newObjectStoreImpl(clusterStore.HTTPRoutes, clusterStore.Updates.HTTPRoutes, logger),
				extractGVK(&v1.Service{}):                      newObjectStoreImpl(clusterStore.Services, clusterStore.Updates.Services, logger),
				extractGVK(&v1.Namespace{}):                    newObjectStoreImpl(clusterStore.Namespaces, clusterStore.Updates.Namespaces, logger),
				extractGVK(&v1.Secret{}):                       newObjectStoreImpl(clusterStore.Secrets, clusterStore.Updates.Secrets, logger),
				extractGVK(&v1.ConfigMap{}):                    newObjectStoreImpl(clusterStore.ConfigMaps, clusterStore.Updates.ConfigMaps, logger),
				extractGVK(&apiext.CustomResourceDefinition{}): newObjectStoreImpl(clusterStore.GatewayAPICRDs, clusterStore.Updates.GatewayAPICRDs, logger),
				extractGVK(&v3.HaproxyGate{}):                  newObjectStoreImpl(clusterStore.HaproxyGates, clusterStore.Updates.HaproxyGates, logger),
				extractGVK(&v3.HugConf{}):                      newObjectStoreImpl(clusterStore.ControllerConfs, clusterStore.Updates.ControllerConfs, logger),
			},
		},
		extractGVK: extractGVK,
		logger:     logger,
	}
}

func (cs *ClusterStoreUpdaterImpl) Upsert(obj client.Object) {
	gvk := cs.extractGVK(obj)
	objectStore, ok := cs.storeAdapter.stores[gvk]
	if !ok {
		return
	}
	objectStore.upsert(obj)
}

func (cs *ClusterStoreUpdaterImpl) Delete(obj client.Object, nsname types.NamespacedName) {
	gvk := cs.extractGVK(obj)
	objectStore, ok := cs.storeAdapter.stores[gvk]
	if !ok {
		return
	}
	objectStore.delete(obj, nsname)
}

func (cs *ClusterStoreUpdaterImpl) ResetUpdates() {
	for _, v := range cs.storeAdapter.stores {
		v.resetUpdates()
	}
}
