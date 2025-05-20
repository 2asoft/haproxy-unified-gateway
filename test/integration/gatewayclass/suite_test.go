// Copyright 2019 HAProxy Technologies LLC
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
	itest "github.com/haproxytech/kubernetes-controller/test/integration"

	"github.com/stretchr/testify/suite"
)

type GatewayClassSuite struct {
	suite.Suite
	test itest.Test
}

func (s *GatewayClassSuite) SetupSuite() {
	var err error
	s.test, err = itest.NewTest(s.T())
	s.Require().NoError(err)

	s.test.StartTestEnv(s.T())
	// defer s.test.StopTestEnv(s.T())
}

func (s *GatewayClassSuite) TearDownSubSuite() {
	s.test.StopTestEnv(s.T())
}

// func TestGatewayClassSuite(t *testing.T) {
// 	suite.Run(t, new(GatewayClassSuite))
// }
