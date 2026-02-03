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

func (s *HostnamesMatchtypeSuite) Test_1_Exact_Route_Exact_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "1-gw-exact-route-exact-match-exact"

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
	expectedMapsPath := path.Join(expectationsPath, "maps")

	// For FE http
	mapFilePath := "link1_" + s.Test().Namespace + "_hug-gateway_http"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	// For FE https
	mapFilePath = "link1_" + s.Test().Namespace + "_hug-gateway_https"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))
}

func (s *HostnamesMatchtypeSuite) Test_5_Wildcard_Route_Exact_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "5-gw-wildcard-route-exact-match-exact"

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

func (s *HostnamesMatchtypeSuite) Test_4_Exact_Route_Empty_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "4-gw-exact-route-empty-match-exact"

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

func (s *HostnamesMatchtypeSuite) Test_2_Exact_Route_Wildcard_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "2-gw-exact-route-wildcard-match-exact"

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

func (s *HostnamesMatchtypeSuite) Test_6_Exact_Route_Exact_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "6-gw-exact-route-exact-match-prefix"

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
