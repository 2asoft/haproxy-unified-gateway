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

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

func (s *HTTPRouteTestSuite) Test_HTTPRoute_BackendCRD_basic() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "backend-crd"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "basic")
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo-backend-cr-basic"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)

	// haproxy.cfg Backends
	backendsExpectationsPath := path.Join(expectationsPath, "backends")
	expectedBackends := []string{"link1_e2e-tests-httproute_http-echo_80_ef84fba452b3f62faf235c47b46c1e54"}
	s.ExpectBackends(s.Test().Ctx, backendsExpectationsPath, expectedBackends)

	// Check Maps
	mapFileRelativePath := "link1_" + s.Test().Namespace + "_gateway_http"
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_BackendCRD_extended() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "backend-crd"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "extended")
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo-backend-cr-basic"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "gateway", "http", 1)

	// haproxy.cfg Backends
	backendsExpectationsPath := path.Join(expectationsPath, "backends")
	expectedBackends := []string{"link1_e2e-tests-httproute_http-echo_80_870306e1a7d334a1eceb23475e6075aa"}
	s.ExpectBackends(s.Test().Ctx, backendsExpectationsPath, expectedBackends)

	// Check Maps
	mapFileRelativePath := "link1_" + s.Test().Namespace + "_gateway_http"
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))
}
