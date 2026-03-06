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

package hostnames_matchtype15

import (
	"testing"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/base"

	"github.com/stretchr/testify/suite"
)

type HostnamesMatchtypeSuite15 struct {
	base.BaseSuite
}

func TestHostnamesMatchtypeSuite15(t *testing.T) {
	suite.Run(t, new(HostnamesMatchtypeSuite15))
}

func (s *HostnamesMatchtypeSuite15) SetupSuite() {
	s.BaseSuite.SetupSuite("../", 0)
}

func (s *HostnamesMatchtypeSuite15) TearDownSuite() {
	s.BaseSuite.TearDownSuite()
}
