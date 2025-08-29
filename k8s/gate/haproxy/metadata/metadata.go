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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
)

const (
	UnifiedGatewayMetatDataKey string = "k8s-unified-ctl"
)

type (
	MetaData map[string]any
)

type K8sObjectInfo struct {
	LinkID     string
	Generation int64
}

type Manager interface {
	FrontendMetaData(treeGw *tree.Gateway) MetaData
}

type ManagerImpl struct {
	extractGVK utils.ExtractGVK
	linkID     string
}

var _ Manager = &ManagerImpl{}

func NewManager(extractGVK utils.ExtractGVK, linkID string) Manager {
	return &ManagerImpl{
		extractGVK: extractGVK,
		linkID:     linkID,
	}
}
