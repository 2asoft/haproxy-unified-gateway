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
package tree

import (
	"log/slog"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
)

type ControllerStore struct {
	ClusterStore      *store.ClusterStore
	GateTree          *GateTree
	UnmanagedGateTree *GateTree
	ReferencedObjects *ReferencedObjects
	// A Map of installed GwApi CRDs versions
	InstalledGwAPIVersions *InstalledVersions
	Logger                 *slog.Logger
	ExtractGVK             utils.ExtractGVK
}

func (b *ControllerStore) CleanInstalledVersionsUpdates() {
	if b.InstalledGwAPIVersions.Updated != nil {
		*b.InstalledGwAPIVersions.Updated = false
	}
}
