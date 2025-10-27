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
	"errors"
	"fmt"
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage/maps"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	k8stypes "k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (b *RouteMgrImpl) processRoutes() error {
	// var err error

	b.fillMaps()
	b.writeMaps()

	if !b.topManager.params.RuntimeUpdateHaproxy {
		return nil
	}

	return errors.New("runtime update not implemented")

	// Process HTTPRoutes
	// b.processHTTPRoutes()

	// b.runtimeCertificatesPrechecks()

	// err := b.executeRuntimeCertCommands()
	// return err
}

type RouteMgrImpl struct {
	topManager *HaproxyConfMgrImpl
}

func (b *RouteMgrImpl) fillMaps() {
	var errs utils.Errors
	controllerStore := b.topManager.controllerStore
	mapsStorage := b.topManager.params.mapsStorage

	// Managed HTTPRoutes => Create / update/ delete backends
	for routeKey, route := range controllerStore.GateTree.HTTPRoutes {
		for _, listener := range route.Listeners.Iterate {
			frontendname, err := b.topManager.getFrontendName(listener.Owner, listener.K8sResource)
			if err != nil {
				b.topManager.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to get frontend name",
					logging.LogAttrError(err),
				)
			}

			pathExactMap := mapsStorage.MapPath(frontendname, storage.PATH_EXACT_MAP)
			pathPrefixMap := mapsStorage.MapPath(frontendname, storage.PATH_PREFIX_MAP)
			pathregexMap := mapsStorage.MapPath(frontendname, storage.PATH_REGEX_MAP)
			mapExact := mapsStorage.GetMapData(pathExactMap)
			mapPrefix := mapsStorage.GetMapData(pathPrefixMap)
			mapRegex := mapsStorage.GetMapData(pathregexMap)

			switch route.TreeStatus.Status {
			case store.StatusUnchanged:
				continue
			case store.StatusUpserted:
				err := b.onUpsertedHTTPRoute(routeKey, route, mapExact, mapPrefix, mapRegex)
				errs.Add(err)
			case store.StatusDeleted:
				err := b.onDeletedHTTPRoute(routeKey, route, mapExact, mapPrefix, mapRegex)
				errs.Add(err)
			}
		}
	}

	// Cleanup Backends that are not referenced anymore TODO
	// if err := b.cleanupUnreferencedBackends(); err != nil {
	// 	b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to cleanup unreferenced backends",
	// 		logging.LogAttrError(err),
	// 	)
	// 	errs.Add(err)
	// }

	// // Now we have the list of upserted + delete BE with the correct list of routes pointing to them
	// if err := b.processBackendsModifiedInCycle(); err != nil {
	// 	b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to process backends modified in cycle",
	// 		logging.LogAttrError(err),
	// 	)
	// 	errs.Add(err)
	// }

	// return errs.Result()
}

func (b *RouteMgrImpl) writeMaps() {
	if !b.topManager.params.StoreMapsOnDisk {
		return
	}

	mapsStorage := b.topManager.params.mapsStorage
	for _, mapData := range mapsStorage.GetMaps() {
		mapsStorage.WriteOnDisk(*mapData)
	}
}

func (b *RouteMgrImpl) onUpsertedHTTPRoute(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute,
	mapExact, mapPrefix, mapRegex *maps.MapData,
) error {
	if route.Valid {
		return b.onValidHTTPRouteUpserted(routeKey, route, mapExact, mapPrefix, mapRegex)
	}
	return b.onInvalidHTTPRouteUpserted(routeKey, route, mapExact, mapPrefix, mapRegex)
}

func (RouteMgrImpl) onDeletedHTTPRoute(_ k8stypes.NamespacedName, route *tree.HTTPRoute,
	// func (b *RouteMgrImpl) onDeletedHTTPRoute(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute,
	mapExact, mapPrefix, mapRegex *maps.MapData,
) error {
	// TODO consider uniting this function with onUpsertedHTTPRoute basically the same
	hostnames := route.K8sResource.Spec.Hostnames
	for _, rule := range route.Rules {
		// if !rule.Valid {
		// find the old rule in route.TreeStatus.OldTreeResource.Rules, name is optional
		// TODO
		// }

		for _, match := range rule.K8sResource.Matches {
			path := "/"
			if match.Path.Value != nil {
				path = *match.Path.Value
			}

			var pathType gatewayv1.PathMatchType
			var mapData *maps.MapData

			if match.Path.Type == nil {
				pathType = gatewayv1.PathMatchPathPrefix
			} else {
				pathType = *match.Path.Type
			}

			switch pathType {
			case gatewayv1.PathMatchExact:
				mapData = mapExact
			case gatewayv1.PathMatchPathPrefix:
				mapData = mapPrefix
			case gatewayv1.PathMatchRegularExpression:
				mapData = mapRegex
			}
			for _, hostname := range hostnames {
				fullpath := string(hostname) + path
				delete(mapData.Data, fullpath)
			}
		}
	}
	return nil
}

func (b *RouteMgrImpl) onValidHTTPRouteUpserted(_ k8stypes.NamespacedName, route *tree.HTTPRoute,
	mapExact, mapPrefix, mapRegex *maps.MapData,
) error { //revive:disable:function-length
	hostnames := route.K8sResource.Spec.Hostnames
	for _, rule := range route.Rules {
		// if !rule.Valid {
		// find the old rule in route.TreeStatus.OldTreeResource.Rules, name is optional
		// TODO
		// }
		var routeValue string
		var backendNames []string
		var backendweights []int32
		for _, backend := range rule.K8sResource.BackendRefs {
			checkResult, ok := rule.CheckBackendRef.Get(backend.BackendObjectReference)
			if !ok || !checkResult.Valid {
				b.topManager.logger.LogAttrs(context.Background(), slog.LevelDebug, "Processing HTTPRoute [map update] - backend not valid",
					logging.LogAttrBackendName(string(backend.Name)),
				)
				continue
			}

			backend := rule.K8sResource.BackendRefs[0]
			svckey := k8stypes.NamespacedName{
				Name: string(backend.Name),
			}
			if backend.Namespace == nil {
				svckey.Namespace = route.K8sResource.Namespace
			} else {
				svckey.Namespace = string(*backend.Namespace)
			}
			svcPort := int32(0)
			if backend.Port != nil {
				svcPort = int32(*backend.Port)
			}
			filterHash := getFilterHash(backend.Filters)
			backendName, err := b.topManager.getBackendName(svckey, int32(svcPort), filterHash)
			if err != nil {
				b.topManager.logger.LogAttrs(context.Background(), slog.LevelError, "Processing HTTPRoute [map update]",
					logging.LogAttrError(err),
				)
				continue
			}
			backendNames = append(backendNames, backendName)
			weight := int32(0)
			if backend.Weight != nil {
				weight = *backend.Weight
			}
			backendweights = append(backendweights, weight)
		}
		if len(backendNames) == 1 {
			routeValue = backendNames[0]
		} else {
			// a: algo, s: suffix, l: list of backend (format depends on algo)
			// /wr_a70_b20_c10     {"a":"wr","l":"a:70,b:20,c:10"}
			routeValue = `{"a":"wr","l":"`
			for i, backendName := range backendNames {
				if i > 0 {
					routeValue += ","
				}
				routeValue += fmt.Sprintf("%s:%d", backendName, backendweights[i])
			}
			routeValue += `"}`
		}

		for _, match := range rule.K8sResource.Matches {
			path := "/"
			if match.Path.Value != nil {
				path = *match.Path.Value
			}

			var pathType gatewayv1.PathMatchType
			var mapData *maps.MapData

			if match.Path.Type == nil {
				pathType = gatewayv1.PathMatchPathPrefix
			} else {
				pathType = *match.Path.Type
			}

			switch pathType {
			case gatewayv1.PathMatchExact:
				mapData = mapExact
			case gatewayv1.PathMatchPathPrefix:
				mapData = mapPrefix
			case gatewayv1.PathMatchRegularExpression:
				mapData = mapRegex
			}

			if rule.Valid {
				for _, hostname := range hostnames {
					fullpath := string(hostname) + path
					mapData.Data[fullpath] = routeValue
				}
			} else {
				for _, hostname := range hostnames {
					fullpath := string(hostname) + path
					delete(mapData.Data, fullpath)
				}
			}
		}
	}

	return nil
}

func (RouteMgrImpl) onInvalidHTTPRouteUpserted(_ k8stypes.NamespacedName, _ *tree.HTTPRoute,
	// func (RouteMgrImpl) onInvalidHTTPRouteUpserted(routeKey k8stypes.NamespacedName, _ *tree.HTTPRoute,
	_, _, _ *maps.MapData,
	// mapExact, mapPrefix, mapRegex *maps.MapData,
) error {
	// TODO we might need to remove it from the maps

	return nil
}
