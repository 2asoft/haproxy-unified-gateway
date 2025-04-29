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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayClassPredicate implements a predicate function based on the controllerName of a GatewayClass.
// This predicate will skip events for GatewayClasses that don't reference this controller.
type GatewayClassPredicate struct {
	predicate.Funcs
	ControllerName string
}

// Create implements default CreateEvent filter for validating a GatewayClass controllerName.
func (gcp GatewayClassPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	gc, ok := e.Object.(*v1.GatewayClass)
	if !ok {
		return false
	}

	return string(gc.Spec.ControllerName) == gcp.ControllerName
}

// Update implements default UpdateEvent filter for validating a GatewayClass controllerName.
func (gcp GatewayClassPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld != nil {
		gcOld, ok := e.ObjectOld.(*v1.GatewayClass)
		if ok && string(gcOld.Spec.ControllerName) == gcp.ControllerName {
			return true
		}
	}

	if e.ObjectNew != nil {
		gcNew, ok := e.ObjectNew.(*v1.GatewayClass)
		if ok && string(gcNew.Spec.ControllerName) == gcp.ControllerName {
			return true
		}
	}

	return false
}

// Delete implements default DeleteEvent filter for validating a GatewayClass controllerName.
func (gcp GatewayClassPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}

	gc, ok := e.Object.(*v1.GatewayClass)
	if !ok {
		return false
	}

	return string(gc.Spec.ControllerName) == gcp.ControllerName
}
