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

	"github.com/Masterminds/semver/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GatewayClassBuilderImpl struct {
	BuilderParams
	gcNames map[string]struct{}
}

var _ Builder = &GatewayClassBuilderImpl{}

type GatewayClassBuilderParams struct {
	BuilderParams
	GcNames map[string]struct{}
}

func NewGatewayClassBuilder(params GatewayClassBuilderParams) *GatewayClassBuilderImpl {
	builder := &GatewayClassBuilderImpl{
		BuilderParams: params.BuilderParams,
		gcNames:       params.GcNames,
	}
	// Register observers
	builder.GateTree.InstalledGwAPIVersions.RegisterObserver(builder.OnUpdateInstalledVersion)

	return builder
}

func (b *GatewayClassBuilderImpl) Build() {
	// First categorize:
	// - accepted
	// - ignored
	categorizer := &GatewayClassCategorizerImpl{gcNames: b.gcNames, gateTree: b.GateTree}
	categorizer.Categorize(b.ClusterStore.Updates.GatewayClasses)

	// Update the references: Gate
	b.updateReferencedGates()

	// Check installed versions
	b.checkSupportedVersion(validateVersionsParams{
		supportedVersions:      SupportedGatewayAPIBundleVersion,
		installedGwAPIVersions: b.GateTree.InstalledGwAPIVersions.Versions,
	})
	// Check Gate reference
	b.checkParametersRef()
}

// OnUpdateInstalledVersion callback function to be called when the installed versions are updated.
func (b *GatewayClassBuilderImpl) OnUpdateInstalledVersion(iv InstalledVersions) {
	b.Logger.LogAttrs(context.Background(), slog.LevelDebug,
		"OnUpdateInstalledVersion",
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrInstalledVersions(iv.Versions),
	)
	// Retrieve Gateway API bundle version
	// using the BundleVersionAnnotation annotation present in all Gateway API CRDs.
	validateVersionsParams := validateVersionsParams{
		supportedVersions:      SupportedGatewayAPIBundleVersion,
		installedGwAPIVersions: b.GateTree.InstalledGwAPIVersions.Versions,
	}
	b.checkSupportedVersion(validateVersionsParams)
	// Update status of all GewayClasses
	for _, gwc := range b.GateTree.GatewayClasses.Supported {
		gwc.buildConditionsSupported(b.GateTree)
	}
	for _, gwc := range b.GateTree.GatewayClasses.Ignored {
		gwc.buildConditionsIgnored(b.GateTree)
	}
}

func (b *GatewayClassBuilderImpl) checkParametersRef() {
	for gwcNsName, gwcUpdate := range b.ClusterStore.Updates.GatewayClasses {
		switch gwcUpdate.Status {
		case store.StatusUpserted:
			gwc := gwcUpdate.NewObject
			gwcTree, ok := b.GateTree.GatewayClasses.Supported[gwcNsName]
			if !ok {
				continue
			}
			paramRef := gwc.Spec.ParametersRef
			checker := HaproxyGateParamsRefChecker{
				ParamRef:          paramRef,
				StoreHaproxyGates: b.ClusterStore.HaproxyGates,
			}
			gwcTree.ParamsRefCheckResult = checker.Check()
		case store.StatusDeleted:
			// nothing to do
		}
	}
}

func (b *GatewayClassBuilderImpl) updateReferencedGates() {
	for _, gwcUpdate := range b.ClusterStore.Updates.GatewayClasses {
		switch gwcUpdate.Status {
		case store.StatusUpserted:
			gwc := gwcUpdate.NewObject
			paramsRef := gwc.Spec.ParametersRef
			if paramsRef == nil {
				continue
			}
			ownedKey := client.ObjectKey{Namespace: utils.NamespaceAsString(paramsRef.Namespace), Name: paramsRef.Name}
			b.GateTree.ReferencedHaproxyGates.AddReferencedBy(ownedKey, gwcUpdate.NewObject)
		case store.StatusDeleted:
			gwc := gwcUpdate.OldObject
			paramsRef := gwc.Spec.ParametersRef
			if paramsRef == nil {
				continue
			}
			ownedKey := client.ObjectKey{Namespace: utils.NamespaceAsString(paramsRef.Namespace), Name: paramsRef.Name}
			b.GateTree.ReferencedHaproxyGates.RemoveReferencedBy(ownedKey, gwcUpdate.OldObject)
		}
	}
}

// Some params to add (reload status, conflicts....)
func (b *GatewayClassBuilderImpl) BuildStatus() {
	for _, gwcUpdate := range b.ClusterStore.Updates.GatewayClasses {
		switch gwcUpdate.Status {
		case store.StatusUpserted:
			gwc := gwcUpdate.NewObject
			var gwcTree *GatewayClass
			var ok bool
			if gwcTree, ok = b.GateTree.GatewayClasses.Supported[client.ObjectKeyFromObject(gwc)]; ok {
				gwcTree.buildConditionsSupported(b.GateTree)
			}
			if gwcTree, ok = b.GateTree.GatewayClasses.Ignored[client.ObjectKeyFromObject(gwc)]; ok {
				gwcTree.buildConditionsIgnored(b.GateTree)
			}
		case store.StatusDeleted:
			// nothing to do
		}
	}
}

type validateVersionsParams struct {
	installedGwAPIVersions map[string]int
	supportedVersions      []string
}

func (b *GatewayClassBuilderImpl) checkSupportedVersion(params validateVersionsParams) {
	for v := range params.installedGwAPIVersions {
		params := validateOneGwAPIVersionParams{
			supportedVersions: params.supportedVersions,
			installedVersion:  v,
		}
		valid := b.validateOneInstalledGwAPIVersion(params)
		if !valid {
			b.GateTree.IsGwAPIVersionValid = false
			return
		}
	}
	b.GateTree.IsGwAPIVersionValid = true
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
			b.Logger.LogAttrs(context.Background(), slog.LevelError,
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
		b.Logger.LogAttrs(context.Background(), slog.LevelError,
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
