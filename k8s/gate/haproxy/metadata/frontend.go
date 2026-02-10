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

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/tree"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FrontendMetaData map[string]map[string]K8sObjectInfo // map[kind] -> map[objectKey]K8sObjectInfo

type GatewayInfo struct {
	gwKey      client.ObjectKey
	generation int64
}

func (mm *ManagerImpl) FrontendMetaData(vListener *tree.VirtualListener) MetaData {
	// Build a list of Gateway info to put in the metadata of the frontend
	infos := make([]GatewayInfo, 0)
	var gvk schema.GroupVersionKind
	for _, listener := range vListener.Listeners {
		treeGw := mm.cs.GetGatewayForListener(listener)
		if treeGw == nil {
			continue
		}
		k8sResource := treeGw.GetK8sResource()
		if k8sResource == nil {
			continue
		}
		// gvk are all the same (Gateway), so we can safely override
		gvk = mm.extractGVK(k8sResource)
		infos = append(infos, GatewayInfo{
			gwKey: client.ObjectKeyFromObject(k8sResource),
			//	gwKind:     gvk.Kind,
			generation: k8sResource.GetGeneration(),
		})
	}

	md := FrontendMetadata(infos, gvk.Kind, mm.linkID)
	return md
}

func FrontendMetadata(infos []GatewayInfo, kind, linkID string) MetaData {
	frontendMetadata := make(FrontendMetaData)

	gatewayMetadata := make(map[string]K8sObjectInfo)
	for _, info := range infos {
		objInfo := K8sObjectInfo{
			Generation: info.generation,
			LinkID:     linkID,
		}
		gatewayMetadata[info.gwKey.String()] = objInfo
	}

	frontendMetadata[kind] = gatewayMetadata

	md := make(MetaData)
	// Gateway metatdata marshall/unmarshal
	// This step is required to ensure that the metadata is in a format that is the same after the one returned from parsing out the metadata from configuration
	o := make(map[string]any)
	by, _ := json.Marshal(frontendMetadata)
	_ = json.Unmarshal(by, &o)

	md[UnifiedGatewayMetaDataKey] = o

	return md
}

// func (mm *ManagerImpl) gatewayKeysFromFrontendMetadata(frontend *models.Frontend) (map[string]struct{}, error) {
// 	frontendMetadataI, ok := frontend.Metadata[UnifiedGatewayMetatDataKey]
// 	gateways := make(map[string]struct{})
// 	if !ok {
// 		return nil, fmt.Errorf("no %s metadata found in frontend %s", UnifiedGatewayMetatDataKey, frontend.Name)
// 	}

// 	by, err := json.Marshal(frontendMetadataI)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var fmd FrontendMetaData
// 	if err := json.Unmarshal(by, &fmd); err != nil {
// 		return nil, fmt.Errorf("failed to unmarshal frontend metadata: %w", err)
// 	}

// 	gatewayMetadata, ok := fmd[b.params.extractGVK(objtypes.ObjectTypeGateway).Kind]
// 	if !ok {
// 		return nil, fmt.Errorf("no %s metadata found in frontend %s", b.params.extractGVK(objtypes.ObjectTypeGateway).Kind, frontend.Name)
// 	}

// 	for key := range gatewayMetadata {
// 		gateways[key] = struct{}{}
// 	}
// 	return gateways, nil
// }
