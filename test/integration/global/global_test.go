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

package global

import (
	"path"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

// Test_Global_Override creates a Global CR with merge_strategy: override and a custom
// maxconn, and verifies that HAProxy picks up the configured value.
func (s *GlobalSuite) Test_Global_Override() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "basic-override")
	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, nil)
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, nil)

	s.expectMaxconn(8192)
}

// Test_Global_Append creates a Global CR with merge_strategy: append and a custom
// maxconn. Scalar fields behave the same as override, so the configured value is applied.
func (s *GlobalSuite) Test_Global_Append() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "basic-append")
	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, nil)
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, nil)

	s.expectMaxconn(8192)
}

// Test_Global_Mode_Override_Replaces_LogTargets verifies that with merge_strategy: override
// the log_target_list from the Global CR fully replaces the default list.
// Default has 1 entry (stdout); CR provides 1 custom entry → result is exactly 1 entry.
func (s *GlobalSuite) Test_Global_Mode_Override_Replaces_LogTargets() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "mode-override")
	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, nil)
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, nil)

	s.expectLogTargetCount(1)
}

// Test_Global_Mode_Append_Extends_LogTargets verifies that with merge_strategy: append
// the log_target_list from the Global CR is appended to the default list.
// Default has 1 entry (stdout); CR adds 1 custom entry → result is 2 entries.
func (s *GlobalSuite) Test_Global_Mode_Append_Extends_LogTargets() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "mode-append")
	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, nil)
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, nil)

	s.expectLogTargetCount(2)
}

// Test_Global_Dynamic_Update verifies that updating a Global CR causes HAProxy to
// pick up the new configuration without restarting.
// Step 1: create Global v1 (maxconn 8192), verify applied.
// Step 2: update to Global v2 (maxconn 16384), verify updated.
func (s *GlobalSuite) Test_Global_Dynamic_Update() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "dynamic-update")

	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf.yaml", "global-v1.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf.yaml", "global-v1.yaml"})

	s.expectMaxconn(8192)

	// Applying global-v2.yaml updates the existing Global CR (same name) to maxconn 16384.
	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, []string{"global-v2.yaml"})
	// Nothing to extra to cleanup as fixture, as global-v2.yaml references the same global,
	// so it is cleaned with the previous CleanupFixturesInNamespace

	s.expectMaxconn(16384)
}

// Test_Global_Dynamic_Switch verifies that updating the HugConf to reference a
// different Global CR causes HAProxy to switch to the new configuration.
// Step 1: HugConf points to global-a (maxconn 8192), verify applied.
// Step 2: HugConf updated to point to global-b (maxconn 16384), verify updated.
func (s *GlobalSuite) Test_Global_Dynamic_Switch() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "dynamic-switch")

	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, []string{"global-a.yaml", "global-b.yaml", "hugconf-a.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, []string{"global-a.yaml", "global-b.yaml", "hugconf-a.yaml"})

	s.expectMaxconn(8192)

	// Applying hugconf-b.yaml updates the existing HugConf (same name) to reference global-b.
	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf-b.yaml"})

	s.expectMaxconn(16384)
}

// Test_Global_Delete_CR verifies that deleting the Global CR causes HAProxy to revert
// to the default global configuration.
// Step 1: create HugConf + Global CR (maxconn 8192), verify applied.
// Step 2: delete Global CR, verify HAProxy reverts to default maxconn (32000).
func (s *GlobalSuite) Test_Global_Delete_CR() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "delete")

	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf.yaml", "global.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf.yaml"})

	s.expectMaxconn(8192)

	// Delete only the Global CR; the HugConf keeps its globalRef but the referenced CR is gone.
	s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, []string{"global.yaml"})

	// HAProxy should revert to the default global maxconn.
	s.expectMaxconn(defaultMaxconn)
}

// Test_Global_Delete_HugConf verifies that deleting the HugConf (which carries the
// globalRef) causes HAProxy to revert to the default global configuration.
func (s *GlobalSuite) Test_Global_Delete_HugConf() {
	fixturePath := path.Join(utils.GetCRDFixturePath(), "delete")

	s.CreateFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf.yaml", "global.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, []string{"global.yaml"})

	s.expectMaxconn(8192)

	// Delete the HugConf: the controller loses the globalRef and marks Global as deleted.
	s.CleanupFixturesInNamespace(fixturePath, hugConfNamespace, []string{"hugconf.yaml"})

	s.expectMaxconn(defaultMaxconn)
}
