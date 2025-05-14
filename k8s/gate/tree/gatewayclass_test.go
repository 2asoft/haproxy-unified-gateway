// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOneInstalledGwApiVersion(t *testing.T) {
	builder := &GatewayClassBuilderImpl{}

	tests := []struct {
		name              string
		supportedVersions []string
		installedVersion  string
		expectedResult    bool
	}{
		{
			name:              "Matching version 1",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion:  "v1.2.5",
			expectedResult:    true,
		},
		{
			name:              "Matching version 2",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion:  "v1.3.5",
			expectedResult:    true,
		},
		{
			name:              "No matching version",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion:  "v2.0.0",
			expectedResult:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := validateOneGwAPIVersionParams{
				supportedVersions: tt.supportedVersions,
				installedVersion:  tt.installedVersion,
			}
			result := builder.validateOneInstalledGwAPIVersion(params)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestValidateVersion(t *testing.T) {
	builder := &GatewayClassBuilderImpl{}

	tests := []struct {
		name              string
		supportedVersions []string
		installedVersion  map[string]struct{}
		expectedResult    bool
	}{
		{
			name:              "Matching version 1",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion: map[string]struct{}{
				"v1.2.5": {},
				"v1.2.4": {},
			},
			expectedResult: true,
		},
		{
			name:              "Matching version 2",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion: map[string]struct{}{
				"v1.3.5": {},
				"v1.3.4": {},
			},
			expectedResult: true,
		},
		{
			name:              "Matching version 1 and 2",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion: map[string]struct{}{
				"v1.2.5": {},
				"v1.3.4": {},
			},
			expectedResult: true,
		},
		{
			name:              "At least one does not match",
			supportedVersions: []string{"v1.2", "v1.3"},
			installedVersion: map[string]struct{}{
				"v1.2.5": {},
				"v1.4.5": {},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.validateVersion(
				validateVersionsParams{
					supportedVersions:      tt.supportedVersions,
					installedGwAPIVersions: tt.installedVersion,
				},
			)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
