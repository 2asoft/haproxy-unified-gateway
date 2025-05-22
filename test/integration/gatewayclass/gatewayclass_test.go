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

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	utils "github.com/haproxytech/kubernetes-controller/test/integration"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

func (s *GatewayClassTestSuite) Test_GatewayClassInvalidParameterRef() {
	fixtureDirPath := utils.GetCRDFixturePath()

	fixturePath := path.Join(fixtureDirPath, "ns_missing")
	err := utils.CreateObjectsFromYAMLFiles(s.Test().Ctx, s.Test().Client, s.Test().Namespace, fixturePath)
	s.Require().NoError(err)

	// Expected Conditions
	expectedConditions := conditions.NewGatewayClassSupportedVersionConditions()
	paramPath := field.NewPath("spec").Child("parametersRef")
	nsPath := paramPath.Child("namespace")
	// notFound := field.NotFound(paramPath, "haproxygate")
	nsrequired := field.Required(nsPath, "namespace is required")
	expectedConditions.MergeOverrideConditions(conditions.NewGatewayClassInvalidParameters(nsrequired))

	s.expectConditionsUpdated(s.Test().Ctx, s.Test().Namespace, "haproxy", expectedConditions)
}
