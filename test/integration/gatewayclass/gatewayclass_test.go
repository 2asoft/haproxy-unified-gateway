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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/constants"
	"github.com/haproxytech/kubernetes-controller/test/integration/utils"
	"github.com/stretchr/testify/suite"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 30
	interval = time.Second * 1
)

// Adding GatewayClassTestSuite, just to be able to debug directly
type GatewayClassTestSuite struct {
	GatewayClassSuite
}

func TestGatewayClassTestSuite(t *testing.T) {
	suite.Run(t, new(GatewayClassTestSuite))
}

func (s *GatewayClassTestSuite) Test_GatewayClass_MissingNamespace() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "nsMissing"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)

	gwcName := "haproxy"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_InvalidRef() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "invalidRef"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)

	gwcName := "haproxy"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_ValidRef() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "validRef"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)

	gwcName := "haproxy"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_Ignored() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "ignored"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	// Expected Conditions
	// For "haproxy" GatewayClass, we expect it to be accepted by the controller
	// and have the "Accepted" condition set to "True".
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)
	gwcName := "haproxy"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions)

	// For "haproxy2" GatewayClass, we expect it to be ignored by the controller
	// and have the "Accepted" condition set to "True".
	expectedCondPath2 := path.Join(expectationsPath, "conditions2.yaml")
	expectedConditions2 := s.YamlToConditions(expectedCondPath2)
	gwcName = "haproxy2"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions2)
}

func (s *GatewayClassTestSuite) Test_GatewayClass_Dynamic_InstalledVersions() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "dynamic-installedversions"
	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath)
	defer s.CleanupFixtures(fixturePath)

	currentVersion := s.setGatewayClassCRToUnsupportedVersion("v1.1")

	// Expected Conditions
	// unsupported version
	expectationsPath := path.Join(fixturePath, "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions-ko.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)
	gwcName := "haproxy"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions)

	// Expected Conditions
	// supported version
	s.resetGatewayClassCRToSupportedVersion(currentVersion)
	expectedCondPath = path.Join(expectationsPath, "conditions-ok.yaml")
	expectedConditions = s.YamlToConditions(expectedCondPath)
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwcName, expectedConditions)
}

func (s *GatewayClassTestSuite) setGatewayClassCRToUnsupportedVersion(newVersion string) string {
	var gatewayClassCRD apiext.CustomResourceDefinition
	err := s.Test().Client.Get(s.Test().Ctx, client.ObjectKey{Name: "gatewayclasses.gateway.networking.k8s.io"}, &gatewayClassCRD)
	s.Require().NoError(err)
	currentVersion := gatewayClassCRD.Annotations[constants.BundleVersionAnnotation]

	gatewayClassCRD.Annotations[constants.BundleVersionAnnotation] = newVersion
	err = s.Test().Client.Update(s.Test().Ctx, &gatewayClassCRD)
	s.Require().NoError(err)

	return currentVersion
}

func (s *GatewayClassTestSuite) resetGatewayClassCRToSupportedVersion(version string) {
	var gatewayClassCRD apiext.CustomResourceDefinition
	err := s.Test().Client.Get(s.Test().Ctx, client.ObjectKey{Name: "gatewayclasses.gateway.networking.k8s.io"}, &gatewayClassCRD)
	s.Require().NoError(err)

	gatewayClassCRD.Annotations[constants.BundleVersionAnnotation] = version
	err = s.Test().Client.Update(s.Test().Ctx, &gatewayClassCRD)
	s.Require().NoError(err)
}
