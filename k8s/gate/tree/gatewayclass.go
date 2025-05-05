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
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type SupportedVersions []string

var SupportedGatewayAPIBundleVersion = SupportedVersions{"v1.1", "v1.2"}

func (s SupportedVersions) String() string {
	return strings.Join(s, ", ")
}

type GatewayClassBuilder interface {
	Build() CategorizedGwApiGatewayClasses
}

type GatewayClassCategorizer interface {
	Categorize(map[types.NamespacedName]*v1.GatewayClass, string) categorizedK8sGatewayClasses
}

type GatewayClassBuilderImpl struct {
	clusterStore *store.ClusterStore
	gcName       string
	categorizer  GatewayClassCategorizer
	logger       *slog.Logger
}

// GatewayClass represents the GatewayClass resource.
type GatewayClass struct {
	// K8sResource is the source resource.
	K8sResource *v1.GatewayClass
	// Conditions include Conditions for the GatewayClass.
	Conditions conditions.Conditions
	// Valid shows whether the GatewayClass is valid.
	Valid bool
}

type categorizedK8sGatewayClasses struct {
	Supported map[types.NamespacedName]*v1.GatewayClass
	Ignored   map[types.NamespacedName]*v1.GatewayClass
}

type CategorizedGwApiGatewayClasses struct {
	Supported map[types.NamespacedName]*GatewayClass
	Ignored   map[types.NamespacedName]*GatewayClass
}

type GatewayClassCategorizerImpl struct{}

var _ GatewayClassBuilder = &GatewayClassBuilderImpl{}

func NewGatewayClassBuilder(
	clusterStore *store.ClusterStore,
	gcName string,
	categorizer GatewayClassCategorizer,
	logger *slog.Logger,
) *GatewayClassBuilderImpl {
	return &GatewayClassBuilderImpl{
		clusterStore: clusterStore,
		gcName:       gcName,
		categorizer:  categorizer,
		logger:       logger,
	}
}

func (builder *GatewayClassBuilderImpl) Build() CategorizedGwApiGatewayClasses {
	categorizedK8sGw := builder.categorizer.Categorize(builder.clusterStore.GatewayClasses, builder.gcName)

	// Retrieve Gateway API bundle version
	// using the BundleVersionAnnotation annotation present in all Gateway API CRDs.
	installedVersions := getGatewayAPIBundleVersions(builder.clusterStore.GatewayApiCRDs)
	versionValid := builder.validateVersion(SupportedGatewayAPIBundleVersion, installedVersions)

	categorizedGwApiGw := CategorizedGwApiGatewayClasses{}

	builder.logger.Info("HELENE", "installedVersions", installedVersions)
	for _, k8sgateway := range categorizedK8sGw.Supported {
		treeGc := GatewayClass{
			K8sResource: k8sgateway,
			Valid:       true,
			Conditions:  conditions.NewDefaultGatewayClassConditions(),
		}
		if !versionValid {
			treeGc.Valid = true
			treeGc.Conditions.MergeOverrideConditions(conditions.NewGatewayClassUnsupportedVersion(SupportedGatewayAPIBundleVersion.String()))
		}
		if categorizedGwApiGw.Supported == nil {
			categorizedGwApiGw.Supported = make(map[types.NamespacedName]*GatewayClass)
		}
		categorizedGwApiGw.Supported[client.ObjectKeyFromObject(k8sgateway)] = &treeGc

	}

	for _, k8sgateway := range categorizedK8sGw.Ignored {
		treeGc := GatewayClass{
			K8sResource: k8sgateway,
			Valid:       false,
			Conditions:  conditions.NewGatewayClassConflict(),
		}
		if categorizedGwApiGw.Ignored == nil {
			categorizedGwApiGw.Ignored = make(map[types.NamespacedName]*GatewayClass)
		}
		categorizedGwApiGw.Ignored[client.ObjectKeyFromObject(k8sgateway)] = &treeGc
	}
	return categorizedGwApiGw
}

func (c *GatewayClassCategorizerImpl) Categorize(
	gatewayClasses map[types.NamespacedName]*v1.GatewayClass,
	gcName string,
) categorizedK8sGatewayClasses {
	filteredGc := categorizedK8sGatewayClasses{}

	for _, gc := range gatewayClasses {
		if gc.Name == gcName {
			if filteredGc.Supported == nil {
				filteredGc.Supported = make(map[types.NamespacedName]*v1.GatewayClass)
			}
			filteredGc.Supported[client.ObjectKeyFromObject(gc)] = gc
		} else {
			if filteredGc.Ignored == nil {
				filteredGc.Ignored = make(map[types.NamespacedName]*v1.GatewayClass)
			}
			filteredGc.Ignored[client.ObjectKeyFromObject(gc)] = gc
		}
	}

	return filteredGc
}

func getGatewayAPIBundleVersions(gatewayAPICRDs map[types.NamespacedName]*metav1.PartialObjectMetadata) map[string]struct{} {
	versions := map[string]struct{}{}

	for _, md := range gatewayAPICRDs {
		bundleVersion := md.Annotations[constants.BundleVersionAnnotation]
		versions[bundleVersion] = struct{}{}
	}
	return versions
}

func (builder *GatewayClassBuilderImpl) validateVersion(supportedVersions []string, installedGwApiVersions map[string]struct{}) bool {
	for v := range installedGwApiVersions {
		valid := builder.validateOneInstalledGwApiVersion(supportedVersions, v)
		if !valid {
			return false
		}
	}
	return true
}

func (builder *GatewayClassBuilderImpl) validateOneInstalledGwApiVersion(supportedVersions []string, installedVersion string) bool {
	constraints := make([]*semver.Constraints, 0)

	for _, v := range supportedVersions {

		constraint, err := semver.NewConstraint("~" + v)
		if err != nil {
			builder.logger.Error("cannot build semver constraint", "error", err)
			return false
		}
		constraints = append(constraints, constraint)
	}

	sv, err := semver.NewVersion(installedVersion)
	if err != nil {
		// If a version string is invalid, we should not consider it as a supported version.
		builder.logger.Error("cannot parse version string", "error", err)
		return false
	}
	for _, constraint := range constraints {
		if constraint.Check(sv) {
			return true
		}
	}

	return false
}
