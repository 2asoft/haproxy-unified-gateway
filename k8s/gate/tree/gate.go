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

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GateBuilderImpl struct {
	BuilderParams
}

var _ Builder = &GateBuilderImpl{}

func (*GateBuilderImpl) Build() {
	// for gateNsName, gateUpdate := range b.ClusterStore.Updates.HaproxyGates {
	// 	// Find the owner: GatewayClasses
	// 	gwcParents := b.GateTree.ReferencedHaproxyGatesgate

	// 	// Find the owner: Gateways
	// }
}

func NewGateBuilder(params BuilderParams) *GateBuilderImpl {
	builder := &GateBuilderImpl{
		BuilderParams: params,
	}
	return builder
}

type HaproxyGateParamsRefChecker struct {
	ParamRef          *v1.ParametersReference
	StoreHaproxyGates map[types.NamespacedName]*v3.HaproxyGate
}

type HaproxyGateParamsRefCheckerResult struct {
	HaproxyGate *v3.HaproxyGate
	Conditions  conditions.Conditions
	Valid       bool
}

func CheckHaproxyGateParamsRef(checker HaproxyGateParamsRefChecker) HaproxyGateParamsRefCheckerResult {
	conds := conditions.Conditions{}
	valid := true
	var haproxyGate *v3.HaproxyGate
	var gateFound bool

	if checker.ParamRef != nil {
		paramPath := field.NewPath("spec").Child("parametersRef")
		if checker.ParamRef.Kind != SupportedGatewayClassParametersRefKind {
			kindPath := paramPath.Child("kind")
			unsupportedKind := field.NotSupported(
				kindPath,
				checker.ParamRef.Kind, []string{string(SupportedGatewayClassParametersRefKind)})
			conds.MergeOverrideConditions(
				conditions.NewGatewayClassInvalidParameters(unsupportedKind),
			)
			valid = false
		} else {
			if checker.ParamRef.Namespace != nil {
				haproxyGate, gateFound = checker.StoreHaproxyGates[types.NamespacedName{
					Name:      checker.ParamRef.Name,
					Namespace: string(*checker.ParamRef.Namespace),
				}]
				if !gateFound {
					notFound := field.NotFound(paramPath, checker.ParamRef.Name)
					conds.MergeOverrideConditions(
						conditions.NewGatewayClassInvalidParameters(notFound),
					)
					valid = false
				}
			} else {
				nsPath := paramPath.Child("namespace")
				nsrequired := field.Required(nsPath, "namespace is required")
				conds.MergeOverrideConditions(
					conditions.NewGatewayClassInvalidParameters(nsrequired),
				)
				valid = false
			}
		}
	}

	return HaproxyGateParamsRefCheckerResult{
		HaproxyGate: haproxyGate,
		Conditions:  conds,
		Valid:       valid,
	}
}

func (*GateBuilderImpl) BuildStatus() {
}
