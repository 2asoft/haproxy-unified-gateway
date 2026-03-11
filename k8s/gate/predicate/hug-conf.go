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
package predicate

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// HugConfPredicate implements a predicate function based on the controller conf CRD ns/name
// This predicate will skip events for GatewayClasses that don't reference this controller.
type HugConfPredicate struct {
	predicate.Funcs
	ControllerConfName types.NamespacedName
}

// Create implements default CreateEvent filter for validating a controller conf CRD ns/name.
func (p HugConfPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}
	objTypesNsName := types.NamespacedName{
		Name:      e.Object.GetName(),
		Namespace: e.Object.GetNamespace(),
	}

	return objTypesNsName == p.ControllerConfName
}

// Update implements default UpdateEvent filter for validating a controller conf CRD ns/name.
func (p HugConfPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld != nil {
		objTypesNsName := types.NamespacedName{
			Name:      e.ObjectOld.GetName(),
			Namespace: e.ObjectOld.GetNamespace(),
		}
		return objTypesNsName == p.ControllerConfName
	}

	if e.ObjectNew != nil {
		objTypesNsName := types.NamespacedName{
			Name:      e.ObjectNew.GetName(),
			Namespace: e.ObjectNew.GetNamespace(),
		}
		return objTypesNsName == p.ControllerConfName
	}

	return false
}

// Delete implements default DeleteEvent filter for validating a controller conf CRD ns/name.
func (p HugConfPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}

	objTypesNsName := types.NamespacedName{
		Name:      e.Object.GetName(),
		Namespace: e.Object.GetNamespace(),
	}
	return objTypesNsName == p.ControllerConfName
}
