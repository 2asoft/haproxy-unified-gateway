//go:build test_todo

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
	"fmt"
	"path"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

func (s *HostnamesMatchtypeSuite) Test_3_Exact_Route_Exact_Match_Exact_Nomatch() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "3-gw-exact-route-exact-math-exact-nomatch"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	//	route := "route-echo-http"
	// TODO fix
	// currently conditions are not udpated
	s.expectRouteConditionsUpdated(s.Test().Ctx, s.Test().Namespace, route, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "hug-gateway", "http", 0)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "hug-gateway", "https", 0)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")

	mapFilePath := "link1_" + s.Test().Namespace + "_hug-gateway_http"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	mapFilePath = "link1_" + s.Test().Namespace + "_hug-gateway_https"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))
}

func (s *HostnamesMatchtypeSuite) Test_7_Exact_Route_Exact_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "7-gw-exact-route-exact-match-regex"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	route := "route-echo-http"
	s.expectRouteConditionsUpdated(s.Test().Ctx, s.Test().Namespace, route, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "hug-gateway", "http", 1)
	s.expectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "hug-gateway", "https", 1)

	// Check Maps
	// TODO fix the maps:
	// path_regexp.map contains:
	// offload.haproxy/^/api/.* link1_test_http-echo_80__
	// This does not work for
	// curl --header "Host: offload.haproxy" http://127.0.0.1:31081/api/foo

	expectedMapsPath := path.Join(expectationsPath, "maps")

	mapFilePath := "link1_" + s.Test().Namespace + "_hug-gateway_http"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	mapFilePath = "link1_" + s.Test().Namespace + "_hug-gateway_https"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))
}
