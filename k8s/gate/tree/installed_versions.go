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
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type SupportedVersions []string

var (
	SupportedGatewayAPIBundleVersion       = SupportedVersions{"v1.2", "v1.3"}
	SupportedGatewayClassParametersRefKind = v1.Kind("HaproxyGate")
)

func (s SupportedVersions) String() string {
	return strings.Join(s, ", ")
}

type InstalledVersionsBuilder interface {
	Build()
}

type InstalledVersionsBuilderImpl struct {
	clusterStore *store.ClusterStore
	tree         *GateTree
	logger       *slog.Logger
}

func NewInstalledVersionsBuilder(clusterStore *store.ClusterStore, tree *GateTree, logger *slog.Logger) *InstalledVersionsBuilderImpl {
	return &InstalledVersionsBuilderImpl{
		clusterStore: clusterStore,
		logger:       logger,
		tree:         tree,
	}
}

func (b *InstalledVersionsBuilderImpl) Build() {
	for nsname, update := range b.clusterStore.Updates.GatewayAPICRDs {
		switch update.Status {
		case store.StatusUpserted:
			b.buildUpserted(nsname, update)
		case store.StatusDeleted:
			b.buildDeleted(update.PreviousObject)
		}
	}
}

func (b *InstalledVersionsBuilderImpl) buildUpserted(nsname types.NamespacedName, update store.Update[*metav1.PartialObjectMetadata]) {
	gwapiCRD, ok := b.clusterStore.GatewayAPICRDs[nsname]
	if !ok {
		err := fmt.Errorf("gwapi CRD not found for %s", nsname)
		b.logger.LogAttrs(
			context.Background(), slog.LevelDebug,
			"gwapi CRD not found",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrError(err),
		)
		return
	}
	bundleVersion := gwapiCRD.Annotations[constants.BundleVersionAnnotation]

	if update.PreviousObject != nil {
		previousBundleVersion := update.PreviousObject.Annotations[constants.BundleVersionAnnotation]
		if previousBundleVersion == bundleVersion {
			return
		}
		b.tree.InstalledGwAPIVersions[previousBundleVersion]--
		if b.tree.InstalledGwAPIVersions[previousBundleVersion] == 0 {
			delete(b.tree.InstalledGwAPIVersions, previousBundleVersion)
		}
	}
	b.tree.InstalledGwAPIVersions[bundleVersion]++
}

func (b *InstalledVersionsBuilderImpl) buildDeleted(previous *metav1.PartialObjectMetadata) {
	// this is wrong, to change
	// implement a ref counter
	bundleVersion := previous.Annotations[constants.BundleVersionAnnotation]
	b.tree.InstalledGwAPIVersions[bundleVersion]--
	if b.tree.InstalledGwAPIVersions[bundleVersion] == 0 {
		delete(b.tree.InstalledGwAPIVersions, bundleVersion)
	}
}
