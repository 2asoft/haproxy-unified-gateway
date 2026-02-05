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

package hostnames_matchtype5

import (
	"testing"
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/base"

	"github.com/stretchr/testify/suite"
)

const (
	timeout  = time.Second * 30
	interval = time.Second * 1
)

type HostnamesMatchtypeSuite5 struct {
	base.BaseSuite
}

func TestHostnamesMatchtypeSuite5(t *testing.T) {
	suite.Run(t, new(HostnamesMatchtypeSuite5))
}

func (s *HostnamesMatchtypeSuite5) SetupSuite() {
	s.BaseSuite.SetupSuite("../", 0)
}

func (s *HostnamesMatchtypeSuite5) TearDownSuite() {
	s.BaseSuite.TearDownSuite()
}
