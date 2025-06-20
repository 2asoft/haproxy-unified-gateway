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

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayClass represents the GatewayClass resource.
type GatewayClass struct {
	// K8sResource is the source resource.
	K8sResource *v1.GatewayClass
	// Conditions include Conditions for the GatewayClass.
	Conditions conditions.Conditions
	// HaproxyGate contains the HaproxyGate (confguration CRD)
	// HaproxyGate *v3.HaproxyGate
	// ParamsRefCheckResult shows whether the GatewayClass is valid as for ParamsRef
	ParamsRefCheckResult HaproxyGateParamsRefCheckResult
	Valid                bool
}

var _ utils.ObjectWithTimestamp = &GatewayClass{}

func NewGatewayClass(k8sObject *v1.GatewayClass) *GatewayClass {
	return &GatewayClass{
		K8sResource: k8sObject,
		Conditions:  conditions.NewDefaultGatewayClassConditions(),
	}
}

func (g *GatewayClass) GetCreationTimestamp() metav1.Time {
	return g.K8sResource.GetCreationTimestamp()
}

func (g *GatewayClass) GetName() string {
	return g.K8sResource.GetName()
}

func (g *GatewayClass) OnGateUpdated(logger *slog.Logger, extractGVK utils.ExtractGVK, updatedGate store.Update[*v3.HaproxyGate],
	clusterStore *store.ClusterStore,
	gateTree *GateTree,
) {
	switch updatedGate.Status {
	case store.StatusUpserted:
		err := checkGateRefConsistency(extractGVK, g, updatedGate.NewObject)
		if err != nil {
			logger.LogAttrs(context.Background(), slog.LevelError,
				"checkGateRefConsistency failed",
				logging.LogAttrCategory(logging.LogCategoryGate),
				logging.LogAttrError(err),
			)
			return
		}
		g.OnGateUpserted(logger, extractGVK, updatedGate.NewObject, clusterStore, gateTree)
	case store.StatusDeleted:
		g.OnGateDeleted(logger, extractGVK, updatedGate.OldObject, clusterStore, gateTree)
	}
}

func (g *GatewayClass) OnGateUpserted(logger *slog.Logger, extractGVK utils.ExtractGVK, gate *v3.HaproxyGate,
	clusterStore *store.ClusterStore,
	gateTree *GateTree,
) {
	gateKey := client.ObjectKeyFromObject(gate)
	logger.LogAttrs(context.Background(), slog.LevelDebug,
		fmt.Sprintf("OnGateUpserted %s", gateKey),
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrResource(g.K8sResource, extractGVK(g.K8sResource)))
	g.BuildConditions(logger, clusterStore, gateTree)
}

func (g *GatewayClass) OnGateDeleted(logger *slog.Logger, extractGVK utils.ExtractGVK, gate *v3.HaproxyGate,
	clusterStore *store.ClusterStore,
	gateTree *GateTree,
) {
	gateKey := client.ObjectKeyFromObject(gate)
	logger.LogAttrs(context.Background(), slog.LevelInfo,
		fmt.Sprintf("OnGateDeleted %s", gateKey),
		logging.LogAttrCategory(logging.LogCategoryGate),
		logging.LogAttrResource(g.K8sResource, extractGVK(g.K8sResource)))
	g.BuildConditions(logger, clusterStore, gateTree)
}

func checkGateRefConsistency(extractGVK utils.ExtractGVK, gwc *GatewayClass, gate *v3.HaproxyGate) error {
	paramRef := gwc.K8sResource.Spec.ParametersRef
	gateGVK := extractGVK(gate)

	if string(paramRef.Group) != gateGVK.Group {
		return fmt.Errorf("group mismatch paramRef %s != gate %s", paramRef.Group, gateGVK.Group)
	}
	if string(paramRef.Kind) != gateGVK.Kind {
		return fmt.Errorf("kind mismatch paramRef %s != gate %s", paramRef.Kind, gateGVK.Kind)
	}
	if paramRef.Namespace != nil {
		if string(*paramRef.Namespace) != gate.Namespace {
			return fmt.Errorf("namespace mismatch paramRef %v != gate %s", paramRef.Namespace, gate.Namespace)
		}
	}
	if paramRef.Name != gate.Name {
		return fmt.Errorf("name mismatch paramRef %s != gate %s", paramRef.Name, gate.Name)
	}

	return nil
}

func (g *GatewayClass) BuildConditions(logger *slog.Logger, clusterStore *store.ClusterStore, gateTree *GateTree) {
	paramRef := g.K8sResource.Spec.ParametersRef
	checker := HaproxyGateParamsRefChecker{
		ParamRef:          paramRef,
		StoreHaproxyGates: clusterStore.HaproxyGates,
	}
	g.ParamsRefCheckResult = checker.Check()
	gwcNsName := client.ObjectKeyFromObject(g.K8sResource)
	// for gwcNsName := range b.ClusterStore.Updates.GatewayClasses {
	isSupported := gateTree.IsSupportedGatewayClass(gwcNsName)
	isIgnored := gateTree.IsIgnoredGatewayClass(gwcNsName)

	if isSupported {
		g.buildConditionsSupported(logger, gateTree)
	}

	if isIgnored {
		g.buildConditionsIgnored(logger, gateTree)
	}
}

func (g *GatewayClass) buildConditionsSupported(_ *slog.Logger, gateTree *GateTree) {
	// Checks on Supported Versions
	switch gateTree.IsGwAPIVersionValid {
	case true:
		g.Conditions.MergeOverrideConditions(
			conditions.NewGatewayClassSupportedVersionConditions())
	case false:
		g.Conditions.MergeOverrideConditions(
			conditions.NewGatewayClassUnsupportedVersion(SupportedGatewayAPIBundleVersion.String()))
	}

	// Checks on parametersRef
	g.Conditions.MergeOverrideConditions(g.ParamsRefCheckResult.Conditions)
	g.Valid = gateTree.IsGwAPIVersionValid && g.ParamsRefCheckResult.Valid
}

func (g *GatewayClass) buildConditionsIgnored(_ *slog.Logger, _ *GateTree) {
	g.Conditions.MergeOverrideConditions(conditions.NewGatewayClassConflict())
}
