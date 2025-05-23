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

package gatewayclass

import (
	"path"
	"testing"
	"time"

	"github.com/haproxytech/kubernetes-controller/test/integration/utils"
	"github.com/stretchr/testify/suite"
)

const (
	timeout  = time.Second * 10
	interval = time.Second * 1
)

// Adding GatewayClassTestSuite, just to be able to debug directly
type GatewayClassTestSuite struct {
	GatewayClassSuite
}

func TestGatewayClassTestSuite(t *testing.T) {
	suite.Run(t, new(GatewayClassTestSuite))
}

func (s *GatewayClassTestSuite) createFixtures(fixturePath string) {
	params := utils.RuntimeYamlParams{
		Ctx:               s.Test().Ctx,
		CrtlruntimeClient: s.Test().Client,
		Namespace:         s.Test().Namespace,
		Dir:               fixturePath,
		WaitForResult:     true,
	}
	err := utils.CreateRuntimeObjectsFromYAMLFiles(params)
	s.Require().NoError(err)
}

func (s *GatewayClassTestSuite) cleanupFixtures(fixturePath string) {
	params := utils.RuntimeYamlParams{
		Ctx:               s.Test().Ctx,
		CrtlruntimeClient: s.Test().Client,
		Namespace:         s.Test().Namespace,
		Dir:               fixturePath,
		WaitForResult:     true,
	}
	err := utils.DeleteRuntimeObjectsFromYAMLFiles(params)
	s.Require().NoError(err)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_MissingNamespace() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "nsMissing"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.createFixtures(fixturePath)
	defer s.cleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)

	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, "haproxy", expectedConditions)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_InvalidRef() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "invalidRef"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.createFixtures(fixturePath)
	defer s.cleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)

	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, "haproxy", expectedConditions)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_ValidRef() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "validRef"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.createFixtures(fixturePath)
	defer s.cleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)

	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, "haproxy", expectedConditions)
}
