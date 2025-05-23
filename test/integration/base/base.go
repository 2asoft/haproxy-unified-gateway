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

package base

import (
	"github.com/stretchr/testify/suite"
)

type BaseSuite struct {
	suite.Suite
	test IntTest
}

func (b *BaseSuite) Test() IntTest {
	return b.test
}

func (b *BaseSuite) SetupSuite() {
	var err error
	b.test, err = NewIntTest(b.T())
	b.Require().NoError(err)

	b.test.StartTestEnv(b.T())
}

func (b *BaseSuite) TearDownSuite() {
	b.test.StopTestEnv(b.T())
}
