//
// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
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
)

type NamespacePredicate struct {
	predicate.Funcs
	namespaces map[string]bool
}

func NewNamespacePredicate(ns []string) NamespacePredicate {
	namespaces := make(map[string]bool, len(ns))
	for _, v := range ns {
		namespaces[v] = true
	}
	return NamespacePredicate{
		namespaces: namespaces,
	}
}

// Create implements default CreateEvent filter for validating a namespace among a list.
func (p NamespacePredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}
	return p.namespaces[e.Object.GetNamespace()]
}

// Update implements default UpdateEvent filter for validating a namespace among a list.
func (p NamespacePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld != nil {
		if p.namespaces[e.ObjectOld.GetNamespace()] {
			return true
		}
	}

	if e.ObjectNew != nil {
		if p.namespaces[e.ObjectNew.GetNamespace()] {
			return true
		}
	}

	return false
}

// Delete implements default DeleteEvent filter for validating a namespace among a list.
func (p NamespacePredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}
	return p.namespaces[e.Object.GetNamespace()]
}
