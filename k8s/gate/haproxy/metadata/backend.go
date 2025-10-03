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
package metadata

import (
	"encoding/json"
)

// map[HTTPRoute] -> map [routeKey]K8sObjectInfo
type BackendMetaData map[string]map[string]K8sObjectInfo // map[kind] -> map[objectKey]K8sObjectInfo

type HTTPRouteMetadaInfo struct {
	OwnerType  string
	Generation int64
}

func (mm *ManagerImpl) BackendMetaData(routesInfo map[string]HTTPRouteMetadaInfo) MetaData {
	md := make(MetaData)

	backendMetadata := make(BackendMetaData)
	for routeKey, routeInfo := range routesInfo {
		_, ok := backendMetadata[routeInfo.OwnerType]
		if !ok {
			backendMetadata[routeInfo.OwnerType] = make(map[string]K8sObjectInfo)
		}
		objInfo := K8sObjectInfo{
			Generation: routeInfo.Generation,
			LinkID:     mm.linkID,
		}
		backendMetadata[routeInfo.OwnerType][routeKey] = objInfo
	}

	// This step is required to ensure that the metadata is in a format that is the same after the one returned from parsing out the metadata from configuration
	o := make(map[string]any)
	by, _ := json.Marshal(backendMetadata)
	_ = json.Unmarshal(by, &o)

	md[UnifiedGatewayMetaDataKey] = o

	return md
}
