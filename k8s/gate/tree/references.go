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
	owner       map[schema.GroupVersionKind]map[client.ObjectKey]struct{}
}

func NewReferencedBy(exctractGVK utils.ExtractGVK) ReferencedBy {
	return ReferencedBy{
		exctractGVK: exctractGVK,
		owner:       make(map[schema.GroupVersionKind]map[client.ObjectKey]struct{}),
	}
}

func (r ReferencedBy) AddReference(owner client.Object) {
	ownerGVK := r.exctractGVK(owner)
	ownerKey := client.ObjectKeyFromObject(owner)
	if _, ok := r.owner[ownerGVK]; !ok {
		r.owner[ownerGVK] = make(map[client.ObjectKey]struct{})
	}
	r.owner[ownerGVK][ownerKey] = struct{}{}
}

func (r ReferencedBy) RemoveReference(owner client.Object) {
	ownerGVK := r.exctractGVK(owner)
	ownerKey := client.ObjectKeyFromObject(owner)
	if _, ok := r.owner[ownerGVK]; !ok {
		return
	}
	delete(r.owner[ownerGVK], ownerKey)
}
