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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/generic"
	rc "github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/routes"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/references"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

type TreeUpdate[T any] struct {
	OldTreeResource *T
	Status          store.Status
}

type Builder interface {
	ComputeTreeUpdates()
	CleanTreeUpdates()
}

// GateTree is a Graph-like representation of Gateway API resources.
type GateTree struct {
	// GatewayClasses holds the GatewayClasses resource that are accepted and ignored
	GatewayClasses map[types.NamespacedName]*GatewayClass
	Gateways       map[types.NamespacedName]*Gateway
	Secrets        map[types.NamespacedName]*Secret
	HTTPRoutes     map[types.NamespacedName]*HTTPRoute
	Services       map[types.NamespacedName]*Service
}

type ReferencedObjects struct {
	// ReferencedHugGates includes the Gates that are references by GatewayClasses and Gateways
	ReferencedHugGates references.ReferencedBy
	//  ReferencedGatewayClasses includes the GatewayClasses that are references by Gateways
	ReferencedGatewayClasses references.ReferencedBy

	//  ReferencedSecrets includes the GatewayClasses that are references by Gateways Listeners
	// Owners are Listeners
	ReferencedSecrets         references.ReferencedBy
	PreviousReferencedSecrets references.ReferencedBy
	// ReferencedGateways includes the Gateways that are referenced by HTTPRoutes
	ReferencedGateways references.ReferencedBy
	// ReferencedServices includes the Services that are references by HTTPRoutes
	ReferencedServices references.ReferencedBy
	// ReferencedBackendCRs includes the Backends CR that are referenced by HTTPRoutes
	ReferencedBackendCRs references.ReferencedBy
}

type CheckResult struct {
	Conditions generic.Conditions
	// If Valid = true, then Conditions should be empty
	// If Valid = false:
	// - Conditions are set if there is an invalid check
	// - Conditions is empty if the check does not make sense (for example no listener status for an invalid Gateway)
	Valid bool
}

type CheckResultRoute struct {
	Conditions rc.RouteConditions
	// If Valid = true, then Conditions should be empty
	// If Valid = false:
	// - Conditions are set if there is an invalid check
	// - Conditions is empty if the check does not make sense (for example no listener status for an invalid Gateway)
	Valid bool
}

func NewGateTree() *GateTree {
	return &GateTree{
		GatewayClasses: make(map[types.NamespacedName]*GatewayClass),
		Gateways:       make(map[types.NamespacedName]*Gateway),
		Secrets:        make(map[types.NamespacedName]*Secret),
		HTTPRoutes:     make(map[types.NamespacedName]*HTTPRoute),
		Services:       make(map[types.NamespacedName]*Service),
	}
}

// TreeResource is a constraint that permits any of the tree's resource types.
type TreeResource interface {
	GatewayClass | Gateway | Secret | HTTPRoute | Service
}

type TreeObject[T TreeResource] interface {
	GetTreeStatus() *TreeUpdate[T]
	SetTreeStatus(TreeUpdate[T])
	DeepCopy() *T
}

// TreeResourcePointer is a constraint for a pointer to a tree resource
// that implements the TreeObject interface.
type TreeResourcePointer[T TreeResource] interface {
	*T
	TreeObject[T]
}

func NewReferencedObjects(extractGVK utils.ExtractGVK) *ReferencedObjects {
	return &ReferencedObjects{
		ReferencedHugGates:        references.NewReferencedBy("huggate", extractGVK),
		ReferencedGatewayClasses:  references.NewReferencedBy("gatewayclass", extractGVK),
		ReferencedSecrets:         references.NewReferencedBy("secret", extractGVK),
		PreviousReferencedSecrets: references.NewReferencedBy("secret", extractGVK),
		ReferencedGateways:        references.NewReferencedBy("gateway", extractGVK),
		ReferencedServices:        references.NewReferencedBy("service", extractGVK),
		ReferencedBackendCRs:      references.NewReferencedBy("backend", extractGVK),
	}
}

func addIndirectFromReferenced[OWNED client.Object, OWNER client.Object](
	ownedUpdate store.Update[OWNED],
	referencedBy references.ReferencedBy,
	ownerMap map[client.ObjectKey]OWNER,
	updateMap map[client.ObjectKey]store.Update[OWNER],
	ownergvk schema.GroupVersionKind,
	ownerKeyTransformer func(client.ObjectKey) client.ObjectKey,
) {
	var owned OWNED
	switch ownedUpdate.Status {
	case store.StatusUpserted:
		owned = ownedUpdate.NewObject
	case store.StatusDeleted:
		owned = ownedUpdate.OldObject
	}

	ownerKeys := referencedBy.ReferencedBy(owned, ownergvk)

	for ownerKey := range ownerKeys {
		if ownerKeyTransformer != nil {
			ownerKey = ownerKeyTransformer(ownerKey)
		}
		owner := ownerMap[ownerKey]
		if _, alreadyPresent := updateMap[ownerKey]; alreadyPresent {
			continue
		}
		updateMap[ownerKey] = store.Update[OWNER]{
			NewObject: owner,
			OldObject: owner, // indirect update
			Status:    store.StatusUpserted,
			Indirect:  true,
		}
	}
}

func cleanTreeUpdates[T TreeResource, R TreeResourcePointer[T]](resourceMap map[types.NamespacedName]R) {
	for key, resource := range resourceMap {
		status := resource.GetTreeStatus()
		if status.Status == store.StatusDeleted {
			delete(resourceMap, key)
			continue
		}
		status.Status = ""
		status.OldTreeResource = nil
	}
}

func setResourceStatus[T TreeResource, R TreeResourcePointer[T]](
	logger *slog.Logger,
	treeResource R,
	logObj client.Object,
	status store.Status,
) {
	logMessage := fmt.Sprintf("%T %s", *treeResource, status)

	logger.LogAttrs(context.Background(), slog.LevelDebug, logMessage,
		logging.LogAttrObjectKey(logObj))

	treeResource.GetTreeStatus().Status = status
	treeResource.GetTreeStatus().OldTreeResource = treeResource.DeepCopy()
}
