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

	semver "github.com/Masterminds/semver/v3"
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayClassBuilder interface {
	Build()
}

type GatewayClassBuilderImpl struct {
	clusterStore        *store.ClusterStore
	tree                *GateTree
	logger              *slog.Logger
	gcNames             map[string]struct{}
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

var _ GatewayClassBuilder = &GatewayClassBuilderImpl{}

type GatewayClassBuilderParams struct {
	ClusterStore *store.ClusterStore
	Tree         *GateTree
	Logger       *slog.Logger
	GcNames      map[string]struct{}
}

func NewGatewayClassBuilder(params GatewayClassBuilderParams) *GatewayClassBuilderImpl {
	builder := &GatewayClassBuilderImpl{
		clusterStore: params.ClusterStore,
		tree:         params.Tree,
		gcNames:      params.GcNames,
		logger:       params.Logger,
	}
	builder.tree.InstalledGwAPIVersions.RegisterObserver(builder.UpdateOnInstalledVersion)

	return builder
	// Register observers
}

func (b *GatewayClassBuilderImpl) Build() {
	// First categorize:
	// - accepted
	// - ignored
	categorizer := &GatewayClassCategorizerImpl{gcNames: b.gcNames, gateTree: b.tree}
	categorizer.Categorize(b.clusterStore.Updates.GatewayClasses)
}

// Some params to add (reload status, conflicts....)
func (b *GatewayClassBuilderImpl) BuildStatus() {
	// Last build conditions
	b.buildConditionsSupportedGwc()
	b.buildConditionsIgnoredGwc()
}

type validateVersionsParams struct {
	installedGwAPIVersions map[string]int
	supportedVersions      []string
}

func (b *GatewayClassBuilderImpl) buildConditionsSupportedGwc() {
	for gwcNsName := range b.clusterStore.Updates.GatewayClasses {
		validVersions := true
		validParamRef := true

		gwc, ok := b.tree.GatewayClasses.Supported[gwcNsName]
		if !ok {
			continue
		}

		gwc.Conditions = conditions.NewDefaultGatewayClassConditions()

		// Checks on Supported Versions
		if !b.isGwAPIVersionValid {
			gwc.Conditions.MergeOverrideConditions(
				conditions.NewGatewayClassUnsupportedVersion(SupportedGatewayAPIBundleVersion.String()))
			validVersions = false
		}

		// Checks on parametersRef
		paramRef := gwc.K8sResource.Spec.ParametersRef
		checker := HaproxyGateParamsRefChecker{
			ParamRef:          paramRef,
			StoreHaproxyGates: b.clusterStore.HaproxyGates,
		}
		refCheckResults := CheckHaproxyGateParamsRef(checker)
		if refCheckResults.Valid {
			gwc.HaproxyGate = refCheckResults.HaproxyGate
		}
		gwc.Conditions.MergeOverrideConditions(refCheckResults.Conditions)
		gwc.Valid = validVersions && validParamRef
	}
}

func (b *GatewayClassBuilderImpl) buildConditionsIgnoredGwc() {
	for gwcNsName := range b.clusterStore.Updates.GatewayClasses {
		gwc, ok := b.tree.GatewayClasses.Ignored[gwcNsName]
		if !ok {
			continue
		}
		gwc.Conditions = conditions.NewGatewayClassConflict()
	}
}

func (b *GatewayClassBuilderImpl) checkSupportedVersion(params validateVersionsParams) {
	for v := range params.installedGwAPIVersions {
		params := validateOneGwAPIVersionParams{
			supportedVersions: params.supportedVersions,
			installedVersion:  v,
		}
		valid := b.validateOneInstalledGwAPIVersion(params)
		if !valid {
			b.isGwAPIVersionValid = false
			return
		}
	}
	b.isGwAPIVersionValid = true
}

type validateOneGwAPIVersionParams struct {
	installedVersion  string
	supportedVersions []string
}

func (b *GatewayClassBuilderImpl) validateOneInstalledGwAPIVersion(params validateOneGwAPIVersionParams) bool {
	constraints := make([]*semver.Constraints, 0)

	for _, v := range params.supportedVersions {
		constraint, err := semver.NewConstraint("~" + v)
		if err != nil {
			b.logger.LogAttrs(context.Background(), slog.LevelError,
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
		b.logger.LogAttrs(context.Background(), slog.LevelError,
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

// UpdateOnInstalledVersion callback function to be called when the installed versions are updated.
func (b *GatewayClassBuilderImpl) UpdateOnInstalledVersion(iv InstalledVersions) {
	b.logger.LogAttrs(context.Background(), slog.LevelDebug,
		"UpdateOnInstalledVersion",
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrInstalledVersions(iv.Versions),
	)
	// Retrieve Gateway API bundle version
	// using the BundleVersionAnnotation annotation present in all Gateway API CRDs.
	validateVersionsParams := validateVersionsParams{
		supportedVersions:      SupportedGatewayAPIBundleVersion,
		installedGwAPIVersions: b.tree.InstalledGwAPIVersions.Versions,
	}
	b.checkSupportedVersion(validateVersionsParams)
}
