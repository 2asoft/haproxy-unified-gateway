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

package gateway

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

// Adding GatewayTestSuite, just to be able to debug directly
type GatewayTestSuite struct {
	GatewaySuite
}

func TestGatewayTestSuite(t *testing.T) {
	suite.Run(t, new(GatewayTestSuite))
}

func (s *GatewayTestSuite) Test_Gateway_InvalidRef() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "invalidRef"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)
	expectedListenerStatusesPath := path.Join(expectationsPath, "listener_statuses.yaml")
	expectedListenerStatuses := s.YamlToListenerStatuses(expectedListenerStatusesPath)

	gwName := "gateway"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwName, expectedConditions, expectedListenerStatuses)
}

func (s *GatewayTestSuite) Test_Gateway_validRef() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "validRef"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)
	expectedListenerStatusesPath := path.Join(expectationsPath, "listener_statuses.yaml")
	expectedListenerStatuses := s.YamlToListenerStatuses(expectedListenerStatusesPath)

	gwName := "gateway"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwName, expectedConditions, expectedListenerStatuses)
}
