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

package hostnames_matchtype3

import (
	"fmt"
	"path"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

func (s *HostnamesMatchtypeSuite3) Test_3_Exact_Route_Exact_Match_Exact_Nomatch() {
	fixtureDirPath := path.Join("../", utils.GetCRDFixturePath())
	fixtureDir := "3-gw-exact-route-exact-match-exact-nomatch"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath, nil)
	mapFilePath2 := "hug_https_31444"
	defer s.Eventually(func() bool {
		correctMapContents := s.CheckMapContents(mapFilePath2, "")
		if !correctMapContents {
			return false
		}
		return s.CheckRuntimeMapContents(mapFilePath2, "")
	}, timeout, interval, fmt.Sprintf("maps in %s were not emptied", mapFilePath2))

	mapFilePath := "hug_http_31081"
	defer s.CleanupFixturesCheckMapFiles(fixturePath, nil, mapFilePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "route-conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	route := "route-echo-http"
	s.ExpectRouteConditionsUpdated(s.Test().Ctx, s.Test().Namespace, route, expectedConditions)

	// Check AttachedRoutes on Gateway status
	s.ExpectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "hug-gateway", "http", 1)
	s.ExpectAttachedRoute(s.Test().Ctx, s.Test().Namespace, "hug-gateway", "https", 1)

	// Check Maps
	expectedMapsPath := path.Join(expectationsPath, "maps")

	// For FE http

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	// For FE https

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}
