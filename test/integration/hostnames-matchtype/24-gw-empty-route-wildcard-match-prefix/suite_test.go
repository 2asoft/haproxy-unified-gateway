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

package hostnames_matchtype24

import (
	"testing"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/base"

	"github.com/stretchr/testify/suite"
)

type HostnamesMatchtypeSuite24 struct {
	base.BaseSuite
}

func TestHostnamesMatchtypeSuite24(t *testing.T) {
	suite.Run(t, new(HostnamesMatchtypeSuite24))
}

func (s *HostnamesMatchtypeSuite24) SetupSuite() {
	s.BaseSuite.SetupSuite("../", 0)
}

func (s *HostnamesMatchtypeSuite24) TearDownSuite() {
	s.BaseSuite.TearDownSuite()
}
