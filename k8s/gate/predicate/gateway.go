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
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GatewayPredicate implements a predicate function based on the gatewayClassName: of a Gateway.
// This predicate will skip events for Gateways that don't reference this gatewayClass.
type GatewayPredicate struct {
	predicate.Funcs
	GatewayClassName string
}

// Create implements default CreateEvent filter for validating a Gateway gatewayClassName.
func (gp GatewayPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	gc, ok := e.Object.(*v1.Gateway)
	if !ok {
		return false
	}

	return string(gc.Spec.GatewayClassName) == gp.GatewayClassName
}

// Update implements default UpdateEvent filter for validating a Gateway gatewayClassName.
func (gp GatewayPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld != nil {
		gcOld, ok := e.ObjectOld.(*v1.Gateway)
		if ok && string(gcOld.Spec.GatewayClassName) == gp.GatewayClassName {
			return true
		}
	}

	if e.ObjectNew != nil {
		gcNew, ok := e.ObjectNew.(*v1.Gateway)
		if ok && string(gcNew.Spec.GatewayClassName) == gp.GatewayClassName {
			return true
		}
	}

	return false
}

// Delete implements default DeleteEvent filter for validating a Gateway gatewayClassName.
func (gp GatewayPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}

	gc, ok := e.Object.(*v1.Gateway)
	if !ok {
		return false
	}

	return string(gc.Spec.GatewayClassName) == gp.GatewayClassName
}
