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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayClassCategorizerImpl_EE struct{}

var _ GatewayClassCategorizer = &GatewayClassCategorizerImpl_EE{}

func (c *GatewayClassCategorizerImpl_EE) Categorize(
	gatewayClasses map[types.NamespacedName]*v1.GatewayClass,
	gcName string,
) categorizedK8sGatewayClasses {
	processedGwClasses := categorizedK8sGatewayClasses{}

	for _, gc := range gatewayClasses {
		if processedGwClasses.Supported == nil {
			processedGwClasses.Supported = make(map[types.NamespacedName]*v1.GatewayClass)
		}
		processedGwClasses.Supported[client.ObjectKeyFromObject(gc)] = gc

	}

	return processedGwClasses
}
