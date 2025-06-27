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
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Gateway represents the Gateway resource.
type Gateway struct {
	// K8sResource is the source resource.
	K8sResource *v1.Gateway
	// Conditions include Conditions for the GatewayClass.
	Conditions conditions.Conditions
	// HaproxyGate contains the HaproxyGate (confguration CRD)
	// The HaproxyGate can be defined at the GatewayClass level or at the Gateway level.
	// Here it is the merged HaproxyGate between the GatewayClass and the Gateway.
	HaproxyGate *v3.HaproxyGate
	// Valid shows whether the GatewayClass is valid.
	Valid bool
}

func NewGateway(k8sObject *v1.Gateway) *Gateway {
	return &Gateway{
		K8sResource: k8sObject,
		Conditions:  conditions.Conditions{},
	}
}
