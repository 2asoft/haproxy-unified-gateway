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
package utils // revive:disable:var-naming

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Helper to create a Gateway with a given name and timestamp offset
func newGateway(name string, now metav1.Time, offset time.Duration) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.Time{Time: now.Add(offset)},
		},
	}
}

func TestSortByCreationTimestamp_Gateways(t *testing.T) {
	now := metav1.Now()
	gateways := []client.Object{
		newGateway("gateway1", now, 1*time.Minute),
		newGateway("gateway2", now, 0),
		newGateway("gateway3", now, 2*time.Minute),
	}

	SortByCreationTimestamp(gateways)

	expectedOrder := []string{"gateway2", "gateway1", "gateway3"}
	for i, gateway := range gateways {
		if gateway.GetName() != expectedOrder[i] {
			t.Errorf("Expected gateway %s at index %d, but got %s", expectedOrder[i], i, gateway.GetName())
		}
	}
}

func TestSortByCreationTimestamp_Gateways_SameTimestamp(t *testing.T) {
	now := metav1.Now()
	gateways := []client.Object{
		newGateway("gateway1b", now, 1*time.Minute),
		newGateway("gateway2", now, 0),
		newGateway("gateway1a", now, 1*time.Minute),
	}

	SortByCreationTimestamp(gateways)

	expectedOrder := []string{"gateway2", "gateway1a", "gateway1b"}
	for i, gateway := range gateways {
		if gateway.GetName() != expectedOrder[i] {
			t.Errorf("Expected gateway %s at index %d, but got %s", expectedOrder[i], i, gateway.GetName())
		}
	}
}
