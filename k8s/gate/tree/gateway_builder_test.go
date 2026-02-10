package tree

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestNbListenersWithAndWithoutConflict(t *testing.T) {
	gwName := "gw1"
	gwNamespace := "default"

	// Helper to create a Gateway
	mkGateway := func(name, namespace string) *Gateway {
		gw := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		return &Gateway{
			K8sResource: gw,
		}
	}

	treeGw := mkGateway(gwName, gwNamespace)

	// Helper to create ObjectKey
	mkKey := func(gwName, listenerName string) client.ObjectKey {
		return client.ObjectKey{
			Namespace: gwNamespace,
			Name:      gwName + "_" + listenerName,
		}
	}

	tests := []struct {
		name                     string
		mapPort2ListenerConflict map[gatewayv1.PortNumber]listenerConflict
		expectedWithConflict     int32
		expectedWithoutConflict  int32
	}{
		{
			name:                     "No listeners",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{},
			expectedWithConflict:     0,
			expectedWithoutConflict:  0,
		},
		{
			name: "Single listener with conflict",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey(gwName, "l1"): {hasConflict: true},
				},
			},
			expectedWithConflict:    1,
			expectedWithoutConflict: 0,
		},
		{
			name: "Single listener without conflict",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey(gwName, "l1"): {hasConflict: false},
				},
			},
			expectedWithConflict:    0,
			expectedWithoutConflict: 1,
		},
		{
			name: "Mixed listeners",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey(gwName, "l1"): {hasConflict: true},
					mkKey(gwName, "l2"): {hasConflict: false},
				},
				443: {
					mkKey(gwName, "l3"): {hasConflict: true},
				},
			},
			expectedWithConflict:    2,
			expectedWithoutConflict: 1,
		},
		{
			name: "Listeners from other gateways ignored",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey(gwName, "l1"):  {hasConflict: true},
					mkKey("other", "l1"): {hasConflict: true},
				},
			},
			expectedWithConflict:    1,
			expectedWithoutConflict: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nb := nbListenersWithAndWithoutConflict(tt.mapPort2ListenerConflict, treeGw)
			assert.Equal(t, tt.expectedWithConflict, nb.withConflict)
			assert.Equal(t, tt.expectedWithoutConflict, nb.withoutConflict)
		})
	}
}

func TestGatewaysWithPortConflicts(t *testing.T) {
	gwNamespace := "default"

	// Helper to create ObjectKey
	mkKey := func(gwName, listenerName string) client.ObjectKey {
		return client.ObjectKey{
			Namespace: gwNamespace,
			Name:      gwName + "_" + listenerName,
		}
	}

	mkGwKey := func(gwName string) client.ObjectKey {
		return client.ObjectKey{
			Namespace: gwNamespace,
			Name:      gwName,
		}
	}

	tests := []struct {
		name                     string
		mapPort2ListenerConflict map[gatewayv1.PortNumber]listenerConflict
		expectedGwKeys           map[client.ObjectKey]struct{}
	}{
		{
			name:                     "No conflicts",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{},
			expectedGwKeys:           map[client.ObjectKey]struct{}{},
		},
		{
			name: "Single gateway with conflict",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey("gw1", "l1"): {hasConflict: true},
				},
			},
			expectedGwKeys: map[client.ObjectKey]struct{}{
				mkGwKey("gw1"): {},
			},
		},
		{
			name: "Single gateway without conflict",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey("gw1", "l1"): {hasConflict: false},
				},
			},
			expectedGwKeys: map[client.ObjectKey]struct{}{},
		},
		{
			name: "Multiple gateways with conflicts",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey("gw1", "l1"): {hasConflict: true},
					mkKey("gw2", "l1"): {hasConflict: true},
				},
			},
			expectedGwKeys: map[client.ObjectKey]struct{}{
				mkGwKey("gw1"): {},
				mkGwKey("gw2"): {},
			},
		},
		{
			name: "Multiple gateways mixed",
			mapPort2ListenerConflict: map[gatewayv1.PortNumber]listenerConflict{
				80: {
					mkKey("gw1", "l1"): {hasConflict: true},
					mkKey("gw2", "l1"): {hasConflict: false},
				},
				443: {
					mkKey("gw3", "l1"): {hasConflict: true},
				},
			},
			expectedGwKeys: map[client.ObjectKey]struct{}{
				mkGwKey("gw1"): {},
				mkGwKey("gw3"): {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gwKeys := gatewaysWithPortConflicts(tt.mapPort2ListenerConflict)
			assert.Equal(t, tt.expectedGwKeys, gwKeys)
		})
	}
}
