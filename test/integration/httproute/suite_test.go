// Copyright 2019 HAProxy Technologies LLC
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

package httproute

import (
	"context"

	"github.com/haproxytech/kubernetes-controller/test/integration/base"
	"github.com/haproxytech/kubernetes-controller/test/integration/utils"

	rc "github.com/haproxytech/kubernetes-controller/k8s/gate/conditions/routes"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type HTTPRouteSuite struct {
	base.BaseSuite
}

func (s *HTTPRouteSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *HTTPRouteSuite) TearDownSuite() {
	s.BaseSuite.TearDownSuite()
}

func (s *HTTPRouteSuite) expectConditionsUpdated(ctx context.Context, namespace, name string, expectedConditions rc.RouteConditions) {
	route := &gatewayv1.HTTPRoute{}
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err := s.Test().Client.Get(
			s.Test().Ctx,
			types.NamespacedName{Name: name, Namespace: namespace}, route); err != nil {
			return false
		}

		gotConditions := rc.NewRouteConditionsFromV1RouteConditions(route.Status, base.TestControllerName)

		res := gotConditions.Equal(expectedConditions)

		return res
	}) {
		s.T().Fatal("conditions not correct")
	}
}
