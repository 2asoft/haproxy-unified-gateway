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
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GateBuilderImpl struct {
	BuilderParams
}

var _ Builder = &GateBuilderImpl{}

func (b *GateBuilderImpl) Build() {
	for _, gateUpdate := range b.ClusterStore.Updates.HaproxyGates {
		var gate *v3.HaproxyGate
		switch gateUpdate.Status {
		case store.StatusUpserted:
			gate = gateUpdate.NewObject
		case store.StatusDeleted:
			gate = gateUpdate.OldObject
		}
		impactedGwcs := FindImpactedGatewayClasses(b.ExtractGVK, gate, *b.ClusterStore, b.GateTree)
		for _, gwc := range impactedGwcs {
			gwc.OnGateUpdated(b.Logger, b.ExtractGVK, gateUpdate, b.ClusterStore, b.GateTree)
		}
	}

	// Find the owner: Gateways
}

func FindImpactedGatewayClasses(extractGVK utils.ExtractGVK, gate *v3.HaproxyGate, clusterStore store.ClusterStore, gateTree *GateTree) []*GatewayClass {
	impactedGateways := make([]*GatewayClass, 0)
	for range clusterStore.Updates.HaproxyGates {
		// Find the owner: GatewayClasses
		gvkGwc := extractGVK(&gatewayv1.GatewayClass{})

		gwcParentKeys := gateTree.ReferencedHaproxyGates.ReferencedBy(gate, gvkGwc)
		for gwcParentKey := range gwcParentKeys {
			// Supported GatewayClasses
			supportedGwcParent, ok := gateTree.GatewayClasses.Supported[gwcParentKey]
			if !ok {
				continue
			}
			impactedGateways = append(impactedGateways, supportedGwcParent)

			// Ignored GatewayClasses
			// Supported GatewayClasses
			ignoredGwcParent, ok := gateTree.GatewayClasses.Ignored[gwcParentKey]
			if !ok {
				continue
			}
			impactedGateways = append(impactedGateways, ignoredGwcParent)
		}
	}
	return impactedGateways
}

func NewGateBuilder(params BuilderParams) *GateBuilderImpl {
	builder := &GateBuilderImpl{
		BuilderParams: params,
	}
	return builder
}

type HaproxyGateParamsRefChecker struct {
	ParamRef          *gatewayv1.ParametersReference
	StoreHaproxyGates map[types.NamespacedName]*v3.HaproxyGate
}

type HaproxyGateParamsRefCheckResult struct {
	// HaproxyGate *v3.HaproxyGate
	Conditions conditions.Conditions
	Valid      bool
}

func (c *HaproxyGateParamsRefChecker) Check() HaproxyGateParamsRefCheckResult {
	conds := conditions.Conditions{}
	valid := true
	// var haproxyGate *v3.HaproxyGate
	var gateFound bool

	if c.ParamRef != nil {
		// Checks that Kind and Group are as expected
		paramPath := field.NewPath("spec").Child("parametersRef")
		if c.ParamRef.Kind != SupportedGatewayClassParametersRefKind {
			kindPath := paramPath.Child("kind")
			unsupportedKind := field.NotSupported(
				kindPath,
				c.ParamRef.Kind, []string{string(SupportedGatewayClassParametersRefKind)})
			conds.MergeOverrideConditions(
				conditions.NewGatewayClassInvalidParameters(unsupportedKind),
			)
			return HaproxyGateParamsRefCheckResult{
				// HaproxyGate: haproxyGate,
				Conditions: conds,
				Valid:      false,
			}
		}
		if c.ParamRef.Group != SupportGatewayClassPamatersRefGroup {
			groupPath := paramPath.Child("group")
			unsupportedGroup := field.NotSupported(
				groupPath,
				c.ParamRef.Group, []string{string(SupportGatewayClassPamatersRefGroup)})
			conds.MergeOverrideConditions(
				conditions.NewGatewayClassInvalidParameters(unsupportedGroup),
			)
			return HaproxyGateParamsRefCheckResult{
				// HaproxyGate: haproxyGate,
				Conditions: conds,
				Valid:      false,
			}
		}
		// Checks that the CR does exist
		if c.ParamRef.Namespace == nil {
			nsPath := paramPath.Child("namespace")
			nsrequired := field.Required(nsPath, "namespace is required")
			conds.MergeOverrideConditions(
				conditions.NewGatewayClassInvalidParameters(nsrequired),
			)
			return HaproxyGateParamsRefCheckResult{
				// HaproxyGate: haproxyGate,
				Conditions: conds,
				Valid:      valid,
			}
		}
		_, gateFound = c.StoreHaproxyGates[types.NamespacedName{
			Name:      c.ParamRef.Name,
			Namespace: string(*c.ParamRef.Namespace),
		}]
		if !gateFound {
			notFound := field.NotFound(paramPath, c.ParamRef.Name)
			conds.MergeOverrideConditions(
				conditions.NewGatewayClassInvalidParameters(notFound),
			)
			return HaproxyGateParamsRefCheckResult{
				//	HaproxyGate: haproxyGate,
				Conditions: conds,
				Valid:      false,
			}
		}
	}
	return HaproxyGateParamsRefCheckResult{
		// HaproxyGate: haproxyGate,
		Conditions: conditions.NewGatewayClassAcceptedConditions(),
		Valid:      valid,
	}
}

func (*GateBuilderImpl) BuildStatus() {
}
