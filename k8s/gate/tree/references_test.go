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
	"testing"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var mockExtractGVK utils.ExtractGVK = func(obj client.Object) schema.GroupVersionKind {
	if _, ok := obj.(*gatewayv1.Gateway); ok {
		return schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"}
	}
	if _, ok := obj.(*gatewayv1.HTTPRoute); ok {
		return schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"}
	}
	return schema.GroupVersionKind{}
}

func TestNewReferencedBy(t *testing.T) {
	rb := NewReferencedBy(mockExtractGVK)
	assert.NotNil(t, rb.owner, "owner map should be initialized")
	assert.NotNil(t, rb.exctractGVK, "exctractGVK function should be set")
}

func TestReferencedBy_AddReference(t *testing.T) {
	rb := NewReferencedBy(mockExtractGVK)

	gw1 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gw1",
			Namespace: "ns1",
		},
	}
	gwGVK := mockExtractGVK(gw1)
	gwKey := client.ObjectKeyFromObject(gw1)

	// Add first reference
	rb.AddReference(gw1)
	assert.Contains(t, rb.owner, gwGVK, "GVK should be added to owner map")
	assert.Contains(t, rb.owner[gwGVK], gwKey, "ObjectKey should be added under GVK")

	// Add another reference with same GVK but different ObjectKey
	gw2 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gw2",
			Namespace: "ns1",
		},
	}
	gw2Key := client.ObjectKeyFromObject(gw2)
	rb.AddReference(gw2)
	assert.Contains(t, rb.owner[gwGVK], gw2Key, "Second ObjectKey should be added under the same GVK")
	assert.Len(t, rb.owner[gwGVK], 2, "Should be two owners for the GVK")

	// Add reference with a different GVK
	hr1 := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hr1",
			Namespace: "ns1",
		},
	}
	hrGVK := mockExtractGVK(hr1)
	hrKey := client.ObjectKeyFromObject(hr1)
	rb.AddReference(hr1)
	assert.Contains(t, rb.owner, hrGVK, "New GVK should be added to owner map")
	assert.Contains(t, rb.owner[hrGVK], hrKey, "ObjectKey should be added under the new GVK")
	assert.Len(t, rb.owner, 2, "Should be two GVKs in the owner map")

	// Add duplicate reference
	rb.AddReference(gw1)
	assert.Len(t, rb.owner[gwGVK], 2, "Adding a duplicate reference should not change the count")
}

func TestReferencedBy_RemoveReference(t *testing.T) {
	rb := NewReferencedBy(mockExtractGVK)

	gw1 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "gw1", Namespace: "ns1"},
	}
	gwGVK := mockExtractGVK(gw1)
	gwKey := client.ObjectKeyFromObject(gw1)

	gw2 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "gw2", Namespace: "ns1"},
	}
	gw2Key := client.ObjectKeyFromObject(gw2)

	hr1 := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "hr1", Namespace: "ns1"},
	}

	// Setup: Add some references
	rb.AddReference(gw1)
	rb.AddReference(gw2)

	// Remove an existing reference
	rb.RemoveReference(gw1)
	assert.Contains(t, rb.owner, gwGVK, "GVK should still exist")
	assert.NotContains(t, rb.owner[gwGVK], gwKey, "gw1 should be removed")
	assert.Contains(t, rb.owner[gwGVK], gw2Key, "gw2 should still exist")
	assert.Len(t, rb.owner[gwGVK], 1, "Only one owner should remain for GVK")

	// Remove the last reference for a GVK
	rb.RemoveReference(gw2)
	assert.Contains(t, rb.owner, gwGVK, "GVK should still exist even if empty")
	assert.NotContains(t, rb.owner[gwGVK], gw2Key, "gw2 should be removed")
	assert.Empty(t, rb.owner[gwGVK], "The map for GVK should be empty")

	// Try to remove a reference that doesn't exist (GVK exists, ObjectKey doesn't)
	rb.AddReference(gw1) // Re-add gw1
	rb.RemoveReference(&gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Name: "nonexistent", Namespace: "ns1"}})
	assert.Contains(t, rb.owner[gwGVK], gwKey, "gw1 should still be present after trying to remove non-existent key")
	assert.Len(t, rb.owner[gwGVK], 1)

	// Try to remove a reference whose GVK doesn't exist
	rb.RemoveReference(hr1) // HTTPRoute GVK was never added
	assert.NotContains(t, rb.owner, mockExtractGVK(hr1), "HTTPRoute GVK should not exist")
	assert.Len(t, rb.owner, 1, "Owner map should still have one GVK (Gateway)")
}
