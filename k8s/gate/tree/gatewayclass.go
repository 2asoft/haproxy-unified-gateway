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
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type GatewayClassBuilder interface {
	Build() CategorizedGatewayClasses
}

type GatewayClassCategorizer interface {
	Categorize(map[types.NamespacedName]*v1.GatewayClass, string) CategorizedGatewayClasses
}

type GatewayClassBuilderImpl struct {
	categorizer GatewayClassCategorizer
	// categorizedK8s      categorizedK8sGatewayClasses
	categorizedGwAPI    CategorizedGatewayClasses
	clusterStore        *store.ClusterStore
	logger              *slog.Logger
	gcName              string
	isGwAPIVersionValid bool
}

// GatewayClass represents the GatewayClass resource.
type GatewayClass struct {
	// K8sResource is the source resource.
	K8sResource *v1.GatewayClass
	// Conditions include Conditions for the GatewayClass.
	Conditions conditions.Conditions
	// HaproxyGate contains the HaproxyGate (confguration CRD)
	HaproxyGate *v3.HaproxyGate
	// Valid shows whether the GatewayClass is valid.
	Valid bool
}

// Supported contains the GatewayClasses that are accepted
// Ignored holds the ignored GatewayClass resources, which reference Haproxy Gateway API controller in
// // `.spec.controllerName`,
// // Those GatewayClass are needed as GatewayAPI spec
// // This is used to update the status of the those GatewayClass resources
type CategorizedGatewayClasses struct {
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

func (builder *GatewayClassBuilderImpl) Build() CategorizedGatewayClasses {
	// First categorize:
	// - accepted
	// - ignored
	builder.categorizedGwAPI = builder.categorizer.Categorize(builder.clusterStore.GatewayClasses, builder.gcName)

	// Retrieve Gateway API bundle version
	// using the BundleVersionAnnotation annotation present in all Gateway API CRDs.
	installedVersions := getGatewayAPIBundleVersions(builder.clusterStore.GatewayAPICRDs)
	builder.logger.LogAttrs(context.Background(), slog.LevelDebug,
		"Installed versions",
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrInstalledVersions(installedVersions),
	)
	validateVersionsParams := validateVersionsParams{
		supportedVersions:      SupportedGatewayAPIBundleVersion,
		installedGwAPIVersions: installedVersions,
	}
	builder.checkSupportedVersion(validateVersionsParams)

	// Last build conditions
	builder.buildConditionsSupportedGwc()
	builder.buildConditionsIgnoredGwc()

	return builder.categorizedGwAPI
}

func (builder *GatewayClassBuilderImpl) buildConditionsSupportedGwc() {
	for _, gwc := range builder.categorizedGwAPI.Supported {
		validVersions := true
		validPramRef := true

		gwc.Conditions = conditions.NewDefaultGatewayClassConditions()

		// Checks on Supported Versions
		if !builder.isGwAPIVersionValid {
			gwc.Conditions.MergeOverrideConditions(
				conditions.NewGatewayClassUnsupportedVersion(SupportedGatewayAPIBundleVersion.String()))
			validVersions = false
		}

		// Checks on parametersRef
		paramRef := gwc.K8sResource.Spec.ParametersRef
		if paramRef != nil {
			paramPath := field.NewPath("spec").Child("parametersRef")
			if paramRef.Kind != SupportedGatewayClassParametersRefKind {
				kindPath := paramPath.Child("kind")
				unsupportedKind := field.NotSupported(
					kindPath,
					paramRef.Kind, []string{string(SupportedGatewayClassParametersRefKind)})
				gwc.Conditions.MergeOverrideConditions(
					conditions.NewGatewayClassInvalidParameters(unsupportedKind),
				)
				validPramRef = false
			} else {
				if paramRef.Namespace != nil {
					haproxyGate, ok := builder.clusterStore.HaproxyGate[types.NamespacedName{
						Name:      paramRef.Name,
						Namespace: string(*paramRef.Namespace),
					}]
					if !ok {
						notFound := field.NotFound(paramPath, paramRef.Name)
						gwc.Conditions.MergeOverrideConditions(
							conditions.NewGatewayClassInvalidParameters(notFound),
						)
						validPramRef = false
					} else {
						gwc.HaproxyGate = haproxyGate
					}
				} else {
					nsPath := paramPath.Child("namespace")
					nsrequired := field.Required(nsPath, "namespace is required")
					gwc.Conditions.MergeOverrideConditions(
						conditions.NewGatewayClassInvalidParameters(nsrequired),
					)
					validPramRef = false
				}
			}

			gwc.Valid = validVersions && validPramRef
		}
	}
}

func (builder *GatewayClassBuilderImpl) buildConditionsIgnoredGwc() {
	for _, gwc := range builder.categorizedGwAPI.Ignored {
		gwc.Conditions = conditions.NewGatewayClassConflict()
	}
}

// CategorizedK8sGatewayClasses is a struct that contains the categorized GatewayClass resources.
// It contains two maps:
// - Supported: GatewayClass resources that are supported by the controller.
// - Ignored: GatewayClass resources that are ignored by the controller.
// For the CE version, gcName shoould not be empty
// Only 1 GatewayClass is supported by the controller.
// The one that has this name
func (*GatewayClassCategorizerImpl) Categorize(
	gatewayClasses map[types.NamespacedName]*v1.GatewayClass, gcName string,
) CategorizedGatewayClasses {
	filteredGc := CategorizedGatewayClasses{}

	for _, gc := range gatewayClasses {
		if gc.Name == gcName {
			if filteredGc.Supported == nil {
				filteredGc.Supported = make(map[types.NamespacedName]*GatewayClass)
			}
			treeGc := GatewayClass{
				K8sResource: gc,
			}
			filteredGc.Supported[client.ObjectKeyFromObject(gc)] = &treeGc
		} else {
			if filteredGc.Ignored == nil {
				filteredGc.Ignored = make(map[types.NamespacedName]*GatewayClass)
			}
			treeGc := GatewayClass{
				K8sResource: gc,
			}
			filteredGc.Ignored[client.ObjectKeyFromObject(gc)] = &treeGc
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

func (builder *GatewayClassBuilderImpl) checkSupportedVersion(params validateVersionsParams) {
	for v := range params.installedGwAPIVersions {
		params := validateOneGwAPIVersionParams{
			supportedVersions: params.supportedVersions,
			installedVersion:  v,
		}
		valid := builder.validateOneInstalledGwAPIVersion(params)
		if !valid {
			builder.isGwAPIVersionValid = false
			return
		}
	}
	builder.isGwAPIVersionValid = true
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
