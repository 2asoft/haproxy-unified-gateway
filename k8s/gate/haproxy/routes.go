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
	"fmt"
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage/maps"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	k8stypes "k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (b *RouteMgrImpl) processRoutes() error {
	// var err error

	// before the sync, clear the dynamic updates
	// we do that before and not later to allow
	// using that data for diff later
	mapsStorage := b.topManager.params.mapsStorage
	for _, mapData := range mapsStorage.GetMaps() {
		mapData.DynamicUpdates.Add = map[string]string{}
		mapData.DynamicUpdates.Update = map[string]string{}
		mapData.DynamicUpdates.Delete = []string{}
	}

	b.fillMaps()
	b.writeMaps()

	if !b.topManager.params.RuntimeUpdateHaproxy {
		return nil
	}

	return b.runtimeMapSync()
}

type RouteMgrImpl struct {
	topManager *HaproxyConfMgrImpl
}

func (b *RouteMgrImpl) onUpsertedHTTPRoute(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute,
	mapExact, mapPrefix, mapRegex, mapDomainWPathExact *maps.MapData,
) error {
	if route.Valid {
		return b.onValidHTTPRouteUpserted(routeKey, route, mapExact, mapPrefix, mapRegex, mapDomainWPathExact)
	}
	return b.onInvalidHTTPRouteUpserted(routeKey, route, mapExact, mapPrefix, mapRegex, mapDomainWPathExact)
}

func (RouteMgrImpl) onDeletedHTTPRoute(_ k8stypes.NamespacedName, route *tree.HTTPRoute,
	// func (b *RouteMgrImpl) onDeletedHTTPRoute(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute,
	mapExact, mapPrefix, mapRegex, mapDomainWPathExact *maps.MapData,
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
				if pathType == gatewayv1.PathMatchExact && isDomainWildcard(string(hostname)) {
					mapDomainWPathExact.DeleteData(fullpath)
				} else {
					mapData.DeleteData(fullpath)
				}
			}
		}
	}
	return nil
}

func (b *RouteMgrImpl) onValidHTTPRouteUpserted(_ k8stypes.NamespacedName, route *tree.HTTPRoute,
	mapExact, mapPrefix, mapRegex, mapDomainWPathExact *maps.MapData,
) error { //revive:disable:function-length,cognitive-complexity
	hostnames := route.K8sResource.Spec.Hostnames
	for _, rule := range route.Rules {
		// if !rule.Valid {
		// find the old rule in route.TreeStatus.OldTreeResource.Rules, name is optional
		// TODO
		// }
		var routeValue string
		var backendNames []string
		var backendweights []int32
		for index, backend := range rule.K8sResource.BackendRefs {
			checkResult, ok := rule.CheckBackendRef.Get(backend.BackendObjectReference)
			if !ok || !checkResult.Valid {
				b.topManager.logger.LogAttrs(context.Background(), slog.LevelDebug, "Processing HTTPRoute [map update] - backend not valid",
					logging.LogAttrBackendName(string(backend.Name)),
				)
				continue
			}

			backend := rule.K8sResource.BackendRefs[index]
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
					if pathType == gatewayv1.PathMatchExact && isDomainWildcard(string(hostname)) {
						mapDomainWPathExact.AddData(fullpath, routeValue)
					} else {
						mapData.AddData(fullpath, routeValue)
						// I need to create a runtime command to add the map entry
					}
				}
			} else {
				for _, hostname := range hostnames {
					fullpath := string(hostname) + path
					if pathType == gatewayv1.PathMatchExact && isDomainWildcard(string(hostname)) {
						mapDomainWPathExact.DeleteData(fullpath)
					} else {
						mapData.DeleteData(fullpath)
					}
				}
			}
		}
	}

	return nil
}

func (RouteMgrImpl) onInvalidHTTPRouteUpserted(_ k8stypes.NamespacedName, _ *tree.HTTPRoute,
	// func (RouteMgrImpl) onInvalidHTTPRouteUpserted(routeKey k8stypes.NamespacedName, _ *tree.HTTPRoute,
	_, _, _, _ *maps.MapData,
	// mapExact, mapPrefix, mapRegex *maps.MapData,
) error {
	// TODO we might need to remove it from the maps

	return nil
}
