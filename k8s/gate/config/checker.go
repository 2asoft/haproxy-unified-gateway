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
package config

import (
	"errors"
)

// Checker is a function to check the configuration.
// This can be a different for EE or CE
func (c *Configuration) Check() error {
	if !c.InitialStructuredHaproxyConfOK {
		return errors.New("initial structured configuration is not set")
	}
	if c.DefaultsSectionName == "" {
		return errors.New("defaults section name is not set")
	}
	if c.LinkID == "" {
		return errors.New("link ID is not set")
	}
	return nil
}
