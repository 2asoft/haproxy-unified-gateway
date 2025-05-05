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
)

// AnnotationPredicate implements a predicate function based on the Annotation.
//
// This predicate will skip the following events:
// 1. Create events that do not contain the Annotation.
// 2. Update events where the Annotation value has not changed.
type AnnotationPredicate struct {
	predicate.Funcs
	Annotation string
}

// Create filters CreateEvents based on the Annotation.
func (cp AnnotationPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	_, ok := e.Object.GetAnnotations()[cp.Annotation]
	return ok
}

// Update filters UpdateEvents based on the Annotation.
func (cp AnnotationPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		// this case should not happen
		return false
	}

	oldAnnotationVal := e.ObjectOld.GetAnnotations()[cp.Annotation]
	newAnnotationVal := e.ObjectNew.GetAnnotations()[cp.Annotation]

	return oldAnnotationVal != newAnnotationVal
}
