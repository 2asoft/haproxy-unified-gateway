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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReferencedBy struct {
	exctractGVK utils.ExtractGVK
	// owner: Gate Key -> GVK of owner -> Owner Key
	owner map[client.ObjectKey]map[schema.GroupVersionKind]map[client.ObjectKey]struct{}
}

func NewReferencedBy(exctractGVK utils.ExtractGVK) ReferencedBy {
	return ReferencedBy{
		exctractGVK: exctractGVK,
		owner:       make(map[client.ObjectKey]map[schema.GroupVersionKind]map[client.ObjectKey]struct{}),
	}
}

func (r ReferencedBy) AddReferencedBy(ownedKey client.ObjectKey, owner client.Object) {
	ownerGVK := r.exctractGVK(owner)
	ownerKey := client.ObjectKeyFromObject(owner)
	if _, ok := r.owner[ownedKey]; !ok {
		r.owner[ownedKey] = make(map[schema.GroupVersionKind]map[client.ObjectKey]struct{})
	}
	if _, ok := r.owner[ownedKey][ownerGVK]; !ok {
		r.owner[ownedKey][ownerGVK] = make(map[client.ObjectKey]struct{})
	}
	r.owner[ownedKey][ownerGVK][ownerKey] = struct{}{}
}

func (r ReferencedBy) RemoveReferencedBy(ownedKey client.ObjectKey, owner client.Object) {
	ownerGVK := r.exctractGVK(owner)
	ownerKey := client.ObjectKeyFromObject(owner)
	if _, ok := r.owner[ownedKey]; !ok {
		return
	}
	if _, ok := r.owner[ownedKey][ownerGVK]; !ok {
		return
	}
	delete(r.owner[ownedKey][ownerGVK], ownerKey)
}

func (r ReferencedBy) ReferencedBy(owned client.Object, ownerGVK schema.GroupVersionKind) map[client.ObjectKey]struct{} {
	ownedKey := client.ObjectKeyFromObject(owned)
	if _, ok := r.owner[ownedKey]; !ok {
		return map[client.ObjectKey]struct{}{}
	}
	if _, ok := r.owner[ownedKey][ownerGVK]; !ok {
		return map[client.ObjectKey]struct{}{}
	}
	return r.owner[ownedKey][ownerGVK]
}
