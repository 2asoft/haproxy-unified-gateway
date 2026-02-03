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

package gatewaytlsmultiple

import (
	"fmt"
	"path"
	"testing"

	futils "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/fileutils"
	"github.com/stretchr/testify/suite"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

func TestGatewayTLSMultipleTestSuite(t *testing.T) {
	suite.Run(t, new(GatewayTLSMultipleSuite))
}

func (s *GatewayTLSMultipleSuite) Test_Gateway_TLS_multiple_same_namespace_ok() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixtureDir := "tls_multiple"

	fixturePath := path.Join(fixtureDirPath, fixtureDir)
	s.CreateFixtures(fixturePath, nil)
	defer s.CleanupFixtures(fixturePath, nil)

	// Expected Conditions
	expectationsPath := path.Join(fixturePath, "okmultiple", "expectations")
	expectedCondPath := path.Join(expectationsPath, "conditions.yaml")
	expectedConditions := s.YamlToConditions(expectedCondPath)
	expectedListenerStatusesPath := path.Join(expectationsPath, "listener_statuses.yaml")
	expectedListenerStatuses := s.YamlToListenerStatuses(expectedListenerStatusesPath)

	gwName := "gateway"
	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, gwName, expectedConditions, expectedListenerStatuses)

	// Check certificates
	expectedCerts := []*models.SslCertificate{
		{
			StorageName: path.Join(s.Test().HaproxyCfgDir, "certs", s.Test().Namespace, "of/e2e-tests-gateway-tls-multiple_offload.pem"),
			Subject:     "/CN=offload.haproxy",
		},
		{
			StorageName: path.Join(s.Test().HaproxyCfgDir, "certs", s.Test().Namespace, "of/e2e-tests-gateway-tls-multiple_offload2.pem"),
			Subject:     "/CN=offload2.haproxy",
		},
	}
	s.ExpectCertificates(s.Test().Ctx, expectedCerts)

	// Check crt-list
	expectedCrtLists := map[futils.FilePath][]string{ // map[crt-list .File]
		{
			Dir:      path.Join(s.Test().HaproxyCfgDir, "certlists"),
			FileName: "/e2e-tests-gateway-tls-multiple_gateway_https.list",
		}: {
			path.Join(s.Test().HaproxyCfgDir, "certs", s.Test().Namespace, "of", "e2e-tests-gateway-tls-multiple_offload.pem"),
			path.Join(s.Test().HaproxyCfgDir, "certs", s.Test().Namespace, "of", "e2e-tests-gateway-tls-multiple_offload2.pem"),
		},
	}
	s.ExpectCrtLists(s.Test().Ctx, expectedCrtLists)

	frontendsExpectationsPath := path.Join(expectationsPath, "frontends")
	expectedFrontends := []string{"link1_e2e-tests-gateway-tls-multiple_gateway_http", "link1_e2e-tests-gateway-tls-multiple_gateway_https"}
	s.ExpectFrontends(s.Test().Ctx, frontendsExpectationsPath, expectedFrontends)

	// Check Maps
	mapFileRelativePath := "link1_" + s.Test().Namespace + "_gateway_http"
	expectedMapsPath := path.Join(expectationsPath, "maps")
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))

	mapFileRelativePath = "link1_" + s.Test().Namespace + "_gateway_https"
	s.Eventually(func() bool {
		return s.CheckMapContents(mapFileRelativePath, expectedMapsPath)
	}, timeout, interval, fmt.Sprintf("maps in %s did not match expected contents", mapFileRelativePath))
}
