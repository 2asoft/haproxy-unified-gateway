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
	"path"
	"testing"
	"time"

	"github.com/haproxytech/kubernetes-controller/test/integration/utils"
	"github.com/stretchr/testify/suite"
)

const (
	timeout  = time.Second * 15
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
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_1_parent_not_allowed() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "1_parent_not_allowed")
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)
}

func (s *HTTPRouteTestSuite) Test_HTTPRoute_no_matching_parent() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "basic"

	fixturePath := path.Join(fixtureDirPath, fixtureDir, "no_matching_parent")
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToRouteConditions(expectedCondPath)

	httpRouteName := "route-echo"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, httpRouteName, expectedConditions)
}
