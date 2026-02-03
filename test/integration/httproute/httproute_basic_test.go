//
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

package httproute

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
	"github.com/stretchr/testify/suite"
)

const (
	timeout  = time.Second * 30
	interval = time.Second * 1
)

// Adding HTTPRouteTestSuite, just to be able to debug directly
type HTTPRouteTestSuite struct {
	HTTPRouteSuite
}

func TestHTTPRouteTestSuite(t *testing.T) {
	suite.Run(t, new(HTTPRouteTestSuite))
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_OK() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "ok")
	s.CreateFixtures(fixturePath, nil)
	mapFileRelativePath2 := "hug_http_8088"
	defer s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath2, "")
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath2, "")
	}, timeout, interval, fmt.Sprintf("maps in %s were not emptied", mapFileRelativePath2))
	mapFileRelativePath := "hug_http_8080"
	defer s.CleanupFixturesCheckMapFiles(fixturePath, nil, mapFileRelativePath)
	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 1)

	// haproxy.cfg Backends
	backendsExpectationsPath := path.Join(expectationsPath, "backends")
	expectedBackends := []string{"hug_e2e-tests-httproute_http-echo_80__"}
	s.ExpectBackends(s.Test().Ctx, backendsExpectationsPath, expectedBackends)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))

	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath2, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath2))
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_1_parent_not_allowed() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"
	fixturePath := path.Join(fixtureDirPath, fixtureDir, "1_parent_not_allowed")
	s.CreateFixtures(fixturePath, nil)
	mapFileRelativePath := "hug_http_8080"
	defer s.CleanupFixturesCheckMapFiles(fixturePath, nil, mapFileRelativePath)
	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 0)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_no_matching_parent() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "no_matching_parent")
	s.CreateFixtures(fixturePath, nil)
	mapFileRelativePath := "hug_http_8080"
	defer s.CleanupFixturesCheckMapFiles(fixturePath, nil, mapFileRelativePath)
	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 0)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_AttachedRoutes() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "attachedroutes"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	mapFileRelativePath := "hug_http_8080"
	s.CreateFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml", "http-echo.yaml", "route.yaml"})
	defer s.CleanupFixturesCheckMapFiles(fixturePath, 
		[]string{"gatewayclass.yaml", "gateway.yaml", "http-echo.yaml", "route.yaml"}, 
		mapFileRelativePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 1)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))

	// 2- Now create a 2nd route
	s.CreateFixtures(fixturePath, []string{"route-2.yaml"})
	expectedCondPath = path.Join(expectationsPath, "route-conditions-2.yaml")
	expectedConditions = s.YamlToRouteConditions(expectedCondPath)

	httpRouteName = "route-echo-2"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 2) // Now 2 routes are attached
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 1)

	// 3- Delete the route "route-echo-2"
	s.CleanupFixtures(fixturePath, []string{"route-2.yaml"})

	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1) // Now back to 1 route attached
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 1)
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_KO_ResolvedRefs() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "ko_resolvedRef")
	s.CreateFixtures(fixturePath, nil)
	mapFileRelativePath := "hug_http_8080"
	defer s.CleanupFixturesCheckMapFiles(fixturePath, nil,mapFileRelativePath)
	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 1)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_OK_Multiple_Listeners_One_Gateway() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "ok _multiple_listeners_one_gateway")
	s.CreateFixtures(fixturePath, nil)
	mapFileRelativePath2 := "hug_http_8088"
	defer s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath2, "")
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath2, "")
	}, timeout, interval, fmt.Sprintf("maps in %s were not emptied", mapFileRelativePath2))
	mapFileRelativePath := "hug_http_8080"
	defer s.CleanupFixturesCheckMapFiles(fixturePath, nil, mapFileRelativePath)
	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http2", 1)

	// haproxy.cfg Backends
	backendsExpectationsPath := path.Join(expectationsPath, "backends")
	expectedBackends := []string{"hug_e2e-tests-httproute_http-echo_80__"}
	s.ExpectBackends(s.Test().Ctx, backendsExpectationsPath, expectedBackends)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))

	s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFileRelativePath2, expectedMapsPath)
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFileRelativePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath2))
}
