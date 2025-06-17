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

var _ Builder = &InstalledVersionsBuilderImpl{}

type InstalledVersions struct {
	// Versions contains the count of installed Gateway API versions.
	Versions  map[string]int // map GwApi CRD version -> counter
	observers []func(InstalledVersions)
	// versionsUpdated bool
}

func (iv *InstalledVersions) RegisterObserver(callback func(InstalledVersions)) {
	iv.observers = append(iv.observers, callback)
}

func (iv *InstalledVersions) NotifyObervers() {
	for _, observer := range iv.observers {
		observer(*iv)
	}
	// iv.versionsUpdated = false
}

func (s SupportedVersions) String() string {
	return strings.Join(s, ", ")
}

type InstalledVersionsBuilderImpl struct {
	BuilderParams
}

func NewInstalledVersionsBuilder(params BuilderParams) *InstalledVersionsBuilderImpl {
	return &InstalledVersionsBuilderImpl{
		BuilderParams: params,
	}
}

func (b *InstalledVersionsBuilderImpl) Build() {
	for nsname, update := range b.ClusterStore.Updates.GatewayAPICRDs {
		switch update.Status {
		case store.StatusUpserted:
			b.buildUpserted(nsname, update)
		case store.StatusDeleted:
			b.buildDeleted(update.OldObject)
		}
	}
	b.Logger.LogAttrs(context.Background(), slog.LevelDebug,
		"Installed versions",
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrInstalledVersions(b.GateTree.InstalledGwAPIVersions.Versions),
	)
	b.GateTree.InstalledGwAPIVersions.NotifyObervers()
}

func (b *InstalledVersionsBuilderImpl) buildUpserted(nsname types.NamespacedName, update store.Update[*metav1.PartialObjectMetadata]) {
	gwapiCRD, ok := b.ClusterStore.GatewayAPICRDs[nsname]
	if !ok {
		err := fmt.Errorf("gwapi CRD not found for %s", nsname)
		b.Logger.LogAttrs(
			context.Background(), slog.LevelDebug,
			"gwapi CRD not found",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrError(err),
		)
		return
	}
	bundleVersion := gwapiCRD.Annotations[constants.BundleVersionAnnotation]

	if update.OldObject != nil {
		previousBundleVersion := update.OldObject.Annotations[constants.BundleVersionAnnotation]
		if previousBundleVersion == bundleVersion {
			return
		}
		b.GateTree.InstalledGwAPIVersions.Versions[previousBundleVersion]--
		// b.GateTree.InstalledGwAPIVersions.versionsUpdated = true
		if b.GateTree.InstalledGwAPIVersions.Versions[previousBundleVersion] == 0 {
			delete(b.GateTree.InstalledGwAPIVersions.Versions, previousBundleVersion)
		}
	}
	// b.GateTree.InstalledGwAPIVersions.versionsUpdated = true
	b.GateTree.InstalledGwAPIVersions.Versions[bundleVersion]++
}

func (b *InstalledVersionsBuilderImpl) buildDeleted(previous *metav1.PartialObjectMetadata) {
	bundleVersion := previous.Annotations[constants.BundleVersionAnnotation]
	b.GateTree.InstalledGwAPIVersions.Versions[bundleVersion]--
	// b.GateTree.InstalledGwAPIVersions.versionsUpdated = true
	if b.GateTree.InstalledGwAPIVersions.Versions[bundleVersion] == 0 {
		delete(b.GateTree.InstalledGwAPIVersions.Versions, bundleVersion)
	}
}

func (*InstalledVersionsBuilderImpl) BuildStatus() {
}
