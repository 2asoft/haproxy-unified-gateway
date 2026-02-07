package diffs

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/types"
)

func TestHaproxyConfResultMarshalJSONWithGatewayObservedGenerations(t *testing.T) {
	result := HaproxyConfResult{
		GatewayObservedGenerations: GatewayObservedGenerations{
			types.NamespacedName{Namespace: "hug-gateways", Name: "shared-gateway"}: 21,
		},
	}

	_, err := json.Marshal(result)

	if err != nil {
		t.Fatalf("json.Marshal(HaproxyConfResult) returned error: %v", err)
	}
}
