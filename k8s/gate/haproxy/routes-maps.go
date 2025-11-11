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
	"log/slog"
	"strings"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/storage"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
)

// ErrMapRuntimeUpdate is an error type for runtime map update failures
type ErrMapRuntimeUpdate struct {
	Err     error
	Type    string
	MapName string
	Key     string
	Value   string
}

func (e *ErrMapRuntimeUpdate) Error() string {
	return "runtime " + e.Type + " map error: map '" + e.MapName + "', key '" + e.Key + "', value '" + e.Value + "': " + e.Err.Error()
}

func (b *RouteMgrImpl) fillMaps() {
	var errs utils.Errors
	controllerStore := b.topManager.controllerStore
	mapsStorage := b.topManager.params.mapsStorage

	// Managed HTTPRoutes => if there is a old resource, clean the state before update
	for routeKey, route := range controllerStore.GateTree.HTTPRoutes {
		if route.TreeStatus.OldTreeResource == nil {
			continue
		}
		route = route.TreeStatus.OldTreeResource
		for _, listener := range route.Listeners.Iterate {
			frontendName, err := b.topManager.getFrontendName(listener.Owner, listener.K8sResource)
			if err != nil {
				b.topManager.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to get frontend name",
					logging.LogAttrError(err),
				)
			}

			pathExactMap := mapsStorage.MapPath(frontendName, storage.PATH_EXACT_MAP)
			pathPrefixMap := mapsStorage.MapPath(frontendName, storage.PATH_PREFIX_MAP)
			pathDomainWPathExactMap := mapsStorage.MapPath(frontendName, storage.PATH_EXACT_DOMAIN_WILDCARD_MAP)
			pathregexMap := mapsStorage.MapPath(frontendName, storage.PATH_REGEX_MAP)
			mapExact := mapsStorage.GetMapData(pathExactMap)
			mapPrefix := mapsStorage.GetMapData(pathPrefixMap)
			mapRegex := mapsStorage.GetMapData(pathregexMap)
			mapDomainWPathExact := mapsStorage.GetMapData(pathDomainWPathExactMap)
			err = b.onDeletedHTTPRoute(routeKey, route, mapExact, mapPrefix, mapRegex, mapDomainWPathExact)
			// errs.Add(err)
			_ = err // TODO ignore error for now
		}
	}

	// Managed HTTPRoutes => Create / update/ delete backends
	for routeKey, route := range controllerStore.GateTree.HTTPRoutes {
		for _, listener := range route.Listeners.Iterate {
			frontendName, err := b.topManager.getFrontendName(listener.Owner, listener.K8sResource)
			if err != nil {
				b.topManager.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to get frontend name",
					logging.LogAttrError(err),
				)
			}

			pathExactMap := mapsStorage.MapPath(frontendName, storage.PATH_EXACT_MAP)
			pathPrefixMap := mapsStorage.MapPath(frontendName, storage.PATH_PREFIX_MAP)
			pathDomainWPathExactMap := mapsStorage.MapPath(frontendName, storage.PATH_EXACT_DOMAIN_WILDCARD_MAP)
			pathRegexMap := mapsStorage.MapPath(frontendName, storage.PATH_REGEX_MAP)
			mapExact := mapsStorage.GetMapData(pathExactMap)
			mapPrefix := mapsStorage.GetMapData(pathPrefixMap)
			mapRegex := mapsStorage.GetMapData(pathRegexMap)
			mapDomainWPathExact := mapsStorage.GetMapData(pathDomainWPathExactMap)

			switch route.TreeStatus.Status {
			case store.StatusUnchanged:
				continue
			case store.StatusUpserted:
				err := b.onUpsertedHTTPRoute(routeKey, route, mapExact, mapPrefix, mapRegex, mapDomainWPathExact)
				errs.Add(err)
			case store.StatusDeleted:
				err := b.onDeletedHTTPRoute(routeKey, route, mapExact, mapPrefix, mapRegex, mapDomainWPathExact)
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

// runtimeMapSync updates the runtime maps through runtime API
func (b *RouteMgrImpl) runtimeMapSync() (mapSyncError error) {
	// runtime check is done before this func is called

	mapsStorage := b.topManager.params.mapsStorage
	runtimeClient := b.topManager.haproxyClient.RuntimeClient()
	for _, mapData := range mapsStorage.GetMaps() {
		// first handle deletions
		for i := len(mapData.DynamicUpdates.Delete) - 1; i >= 0; i-- {
			key := mapData.DynamicUpdates.Delete[i]
			mapID := b.getMapID(mapData.Path.FullPath())
			if mapID == "" {
				return &ErrMapRuntimeUpdate{
					Type:    "update",
					MapName: mapData.Path.FullPath(),
					Key:     key,
					Err:     errors.New("map not found"),
				}
			}
			err := runtimeClient.DeleteMapEntry(mapID, key)
			if err != nil {
				return &ErrMapRuntimeUpdate{
					Type:    "delete",
					MapName: mapData.Path.FileName,
					Key:     key,
					Err:     err,
				}
			}
		}

		// then handle additions / updates
		for key, val := range mapData.DynamicUpdates.Update {
			mapID := b.getMapID(mapData.Path.FullPath())
			if mapID == "" {
				return &ErrMapRuntimeUpdate{
					Type:    "update",
					MapName: mapData.Path.FullPath(),
					Key:     key,
					Value:   val,
					Err:     errors.New("map not found"),
				}
			}
			err := runtimeClient.SetMapEntry(mapID, key, val)
			if err != nil {
				return &ErrMapRuntimeUpdate{
					Type:    "update",
					MapName: mapData.Path.FileName,
					Key:     key,
					Value:   val,
					Err:     err,
				}
			}
		}

		for key, val := range mapData.DynamicUpdates.Add {
			mapID := b.getMapID(mapData.Path.FullPath())
			if mapID == "" {
				return &ErrMapRuntimeUpdate{
					Type:    "add",
					MapName: mapData.Path.FullPath(),
					Key:     key,
					Value:   val,
					Err:     errors.New("map not found"),
				}
			}
			err := runtimeClient.AddMapEntry(mapID, key, val)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				return &ErrMapRuntimeUpdate{
					Type:    "add",
					MapName: mapData.Path.FileName,
					Key:     key,
					Value:   val,
					Err:     err,
				}
			}
		}
	}
	return nil
}

func (b *RouteMgrImpl) getMapID(fullPath string) string {
	runtimeClient := b.topManager.haproxyClient.RuntimeClient()
	// find the id of a map entry
	maps, _ := runtimeClient.ShowMaps()
	for _, m := range maps {
		if m.File == fullPath {
			return "#" + m.ID
		}
	}
	return ""
}
