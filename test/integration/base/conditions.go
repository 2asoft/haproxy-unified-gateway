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

package base

import (
	"os"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (b *BaseSuite) YamlToConditions(yamlPath string) conditions.Conditions {
	yamlFile, err := os.ReadFile(yamlPath)
	assert.NoError(b.T(), err, "Failed to read YAML file")

	var expectedConditionsMetaV1 []metav1.Condition
	err = yaml.Unmarshal(yamlFile, &expectedConditionsMetaV1)
	assert.NoError(b.T(), err, "Failed to unmarshal YAML")
	return conditions.NewConditionsFromMetav1Conditions(expectedConditionsMetaV1)
}
