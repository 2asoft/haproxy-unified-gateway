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
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

const (
	timeout  = time.Second * 15
	interval = time.Second * 1
)

func (s *HostnamesMatchtypeSuite) Test_1_Exact_Route_Exact_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "1-gw-exact-route-exact-match-exact"
	mapFilePath := "hug_http_31081"
	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath, nil)
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
	mapFilePath = "hug_https_31444"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))
}

func (s *HostnamesMatchtypeSuite) Test_5_Wildcard_Route_Exact_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "5-gw-wildcard-route-exact-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_4_Exact_Route_Empty_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "4-gw-exact-route-empty-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_2_Exact_Route_Wildcard_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "2-gw-exact-route-wildcard-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_6_Exact_Route_Exact_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "6-gw-exact-route-exact-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_8_Exact_Route_Wildcard_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "8-gw-exact-route-wildcard-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_10_Exact_Route_Empty_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "10-gw-exact-route-empty-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_12_Wildcard_Route_Exact_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "12-gw-wildcard-route-exact-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_14_Wildcard_Route_Wildcard_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "14-gw-wildcard-route-wildcard-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_17_Wildcard_Route_Empty_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "17-gw-wildcard-route-empty-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_20_Empty_Route_Exact_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "20-gw-empty-route-exact-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_23_Empty_Route_Wildcard_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "23-gw-empty-route-wildcard-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_26_Empty_Route_Empty_Match_Exact() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "26-gw-empty-route-empty-match-exact"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_27_Empty_Route_Empty_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "27-gw-empty-route-empty-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_13_Wildcard_Route_Exact_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "13-gw-wildcard-route-exact-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_7_Exact_Route_Exact_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "7-gw-exact-route-exact-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_9_Exact_Route_Wildcard_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "9-gw-exact-route-wildcard-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_11_Exact_Route_Empty_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "11-gw-exact-route-empty-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_16_Wildcard_Route_Wildcard_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "16-gw-wildcard-route-wildcard-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_19_Wildcard_Route_Empty_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "19-gw-wildcard-route-empty-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_25_Empty_Route_Wildcard_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "25-gw-empty-route-wildcard-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_28_Empty_Route_Empty_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "28-gw-empty-route-empty-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_22_Empty_Route_Exact_Match_Regex() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "22-gw-empty-route-exact-match-regex"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_15_Wildcard_Route_Wildcard_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "15-gw-wildcard-route-wildcard-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_18_Wildcard_Route_Empty_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "18-gw-wildcard-route-empty-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_21_Empty_Route_Exact_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "21-gw-empty-route-exact-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}

func (s *HostnamesMatchtypeSuite) Test_24_Empty_Route_Wildcard_Match_Prefix() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "24-gw-empty-route-wildcard-match-prefix"

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

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath))

	s.Eventually(func() bool {
		return s.CheckMapContents(mapFilePath2, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFilePath2))
}
