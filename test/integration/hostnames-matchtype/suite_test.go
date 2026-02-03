// Copyright 2025 HAProxy Technologies LLC
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

package hostnames_matchtype

import (
	"context"
	"testing"
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/base"
	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"

	rc "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/conditions/routes"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	timeout  = time.Second * 30
	interval = time.Second * 1
)

type HostnamesMatchtypeSuite struct {
	base.BaseSuite
}

func TestHostnamesMatchtypeSuite(t *testing.T) {
	suite.Run(t, new(HostnamesMatchtypeSuite))
}

func (s *HostnamesMatchtypeSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *HostnamesMatchtypeSuite) TearDownSuite() {
	s.BaseSuite.TearDownSuite()
}

func (s *HostnamesMatchtypeSuite) expectRouteConditionsUpdated(ctx context.Context, namespace, name string, expectedConditions rc.RouteConditions) {
	route := &gatewayv1.HTTPRoute{}
	var gotConditions rc.RouteConditions
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err := s.Test().Client.Get(
			s.Test().Ctx,
			types.NamespacedName{Name: name, Namespace: namespace}, route); err != nil {
			return false
		}

		gotConditions = rc.NewRouteConditionsFromV1RouteConditions(route.Status.Parents, base.TestControllerName)

		res := gotConditions.Equal(expectedConditions)

		return res
	}) {
		s.T().Fatalf("conditions not correct,\nGot %+v\nExpected %+v\n", gotConditions, expectedConditions)
	}
}

func (s *HostnamesMatchtypeSuite) expectAttachedRoute(ctx context.Context, namespace, gwName, listenerName string, expectNbAttachedRoutes int32) {
	gw := &gatewayv1.Gateway{}
	if !utils.WaitFor(ctx, interval, timeout, func() bool {
		if err := s.Test().Client.Get(
			s.Test().Ctx,
			types.NamespacedName{Name: gwName, Namespace: namespace}, gw); err != nil {
			return false
		}

		for _, listenerStatus := range gw.Status.Listeners {
			if string(listenerStatus.Name) == listenerName {
				return listenerStatus.AttachedRoutes == expectNbAttachedRoutes
			}
		}

		return false
	}) {
		s.T().Fatal("AttachedRoutes not correct")
	}
}
