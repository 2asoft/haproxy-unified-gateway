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
	"log/slog"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type SupportedVersions []string

var SupportedGatewayAPIBundleVersion = SupportedVersions{"v1.2", "v1.3"}

func (s SupportedVersions) String() string {
	return strings.Join(s, ", ")
}

type GatewayClassBuilder interface {
	Build() CategorizedGwAPIGatewayClasses
}

type GatewayClassCategorizer interface {
	Categorize(map[types.NamespacedName]*v1.GatewayClass, string) categorizedK8sGatewayClasses
}

type GatewayClassBuilderImpl struct {
	categorizer  GatewayClassCategorizer
	clusterStore *store.ClusterStore
	logger       *slog.Logger
	gcName       string
}

// GatewayClass represents the GatewayClass resource.
type GatewayClass struct {
	// K8sResource is the source resource.
	K8sResource *v1.GatewayClass
	// Conditions include Conditions for the GatewayClass.
	Conditions conditions.Conditions
	v1.Gateway
	// Valid shows whether the GatewayClass is valid.
	Valid bool
}

type categorizedK8sGatewayClasses struct {
	Supported map[types.NamespacedName]*v1.GatewayClass
	Ignored   map[types.NamespacedName]*v1.GatewayClass
}

type CategorizedGwAPIGatewayClasses struct {
	Supported map[types.NamespacedName]*GatewayClass
	Ignored   map[types.NamespacedName]*GatewayClass
}

type GatewayClassCategorizerImpl struct{}

var _ GatewayClassBuilder = &GatewayClassBuilderImpl{}

type GatewayClassBuilderParams struct {
	Categorizer  GatewayClassCategorizer
	ClusterStore *store.ClusterStore
	Logger       *slog.Logger
	GcName       string
}

func NewGatewayClassBuilder(params GatewayClassBuilderParams) *GatewayClassBuilderImpl {
	return &GatewayClassBuilderImpl{
		clusterStore: params.ClusterStore,
		gcName:       params.GcName,
		categorizer:  params.Categorizer,
		logger:       params.Logger,
	}
}

func (builder *GatewayClassBuilderImpl) Build() CategorizedGwAPIGatewayClasses {
	categorizedK8sGw := builder.categorizer.Categorize(builder.clusterStore.GatewayClasses, builder.gcName)

	// Retrieve Gateway API bundle version
	// using the BundleVersionAnnotation annotation present in all Gateway API CRDs.
	installedVersions := getGatewayAPIBundleVersions(builder.clusterStore.GatewayAPICRDs)
	validateVersionsParams := validateVersionsParams{
		supportedVersions:      SupportedGatewayAPIBundleVersion,
		installedGwAPIVersions: installedVersions,
	}
	versionValid := builder.validateVersion(validateVersionsParams)

	categorizedGwAPIGw := CategorizedGwAPIGatewayClasses{}

	builder.logger.LogAttrs(context.Background(), slog.LevelDebug,
		"",
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrInstalledVersions(installedVersions),
	)
	for _, k8sgateway := range categorizedK8sGw.Supported {
		treeGc := GatewayClass{
			K8sResource: k8sgateway,
			Valid:       true,
			Conditions:  conditions.NewDefaultGatewayClassConditions(),
		}
		if !versionValid {
			treeGc.Valid = true
			treeGc.Conditions.MergeOverrideConditions(
				conditions.NewGatewayClassUnsupportedVersion(SupportedGatewayAPIBundleVersion.String()))
		}
		if categorizedGwAPIGw.Supported == nil {
			categorizedGwAPIGw.Supported = make(map[types.NamespacedName]*GatewayClass)
		}
		categorizedGwAPIGw.Supported[client.ObjectKeyFromObject(k8sgateway)] = &treeGc
	}

	for _, k8sgateway := range categorizedK8sGw.Ignored {
		treeGc := GatewayClass{
			K8sResource: k8sgateway,
			Valid:       false,
			Conditions:  conditions.NewGatewayClassConflict(),
		}
		if categorizedGwAPIGw.Ignored == nil {
			categorizedGwAPIGw.Ignored = make(map[types.NamespacedName]*GatewayClass)
		}
		categorizedGwAPIGw.Ignored[client.ObjectKeyFromObject(k8sgateway)] = &treeGc
	}
	return categorizedGwAPIGw
}

// CategorizedK8sGatewayClasses is a struct that contains the categorized GatewayClass resources.
// It contains two maps:
// - Supported: GatewayClass resources that are supported by the controller.
// - Ignored: GatewayClass resources that are ignored by the controller.
// For the CE version, gcName shoould not be empty
// Only 1 GatewayClass is supported by the controller.
// The one that has this name
func (*GatewayClassCategorizerImpl) Categorize(
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

type installedGwAPIVersions map[types.NamespacedName]*metav1.PartialObjectMetadata

func getGatewayAPIBundleVersions(gatewayAPICRDs installedGwAPIVersions) map[string]struct{} {
	versions := map[string]struct{}{}

	for _, md := range gatewayAPICRDs {
		bundleVersion := md.Annotations[constants.BundleVersionAnnotation]
		versions[bundleVersion] = struct{}{}
	}
	return versions
}

type validateVersionsParams struct {
	installedGwAPIVersions map[string]struct{}
	supportedVersions      []string
}

func (builder *GatewayClassBuilderImpl) validateVersion(params validateVersionsParams) bool {
	for v := range params.installedGwAPIVersions {
		params := validateOneGwAPIVersionParams{
			supportedVersions: params.supportedVersions,
			installedVersion:  v,
		}
		valid := builder.validateOneInstalledGwAPIVersion(params)
		if !valid {
			return false
		}
	}
	return true
}

type validateOneGwAPIVersionParams struct {
	installedVersion  string
	supportedVersions []string
}

func (builder *GatewayClassBuilderImpl) validateOneInstalledGwAPIVersion(params validateOneGwAPIVersionParams) bool {
	constraints := make([]*semver.Constraints, 0)

	for _, v := range params.supportedVersions {
		constraint, err := semver.NewConstraint("~" + v)
		if err != nil {
			builder.logger.LogAttrs(context.Background(), slog.LevelError,
				"cannot build semver constraint",
				logging.LogAttrCategory(logging.LogCategoryGate),
				logging.LogAttrError(err),
			)
			return false
		}
		constraints = append(constraints, constraint)
	}

	sv, err := semver.NewVersion(params.installedVersion)
	if err != nil {
		// If a version string is invalid, we should not consider it as a supported version.
		builder.logger.LogAttrs(context.Background(), slog.LevelError,
			"cannot parse version string",
			logging.LogAttrCategory(logging.LogCategoryGate),
			logging.LogAttrError(err),
		)
		return false
	}
	for _, constraint := range constraints {
		if constraint.Check(sv) {
			return true
		}
	}

	return false
}

var _ utils.ObjectWithTimestamp = &GatewayClass{}

func (g *GatewayClass) GetCreationTimestamp() metav1.Time {
	return g.K8sResource.GetCreationTimestamp()
}

func (g *GatewayClass) GetName() string {
	return g.K8sResource.GetName()
}
