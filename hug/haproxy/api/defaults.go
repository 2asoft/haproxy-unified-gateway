// Copyright 2025 HAProxy Technologies LLC
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
package api

import (
	"encoding/json"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/hug/reload"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/constants"
	"github.com/imdario/mergo"
)

func (c *clientNative) DefaultsSectionGet(name string) (*models.Defaults, error) {
	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return nil, err
	}
	_, defaults, err := configuration.GetStructuredDefaultsSection(name, c.activeTransaction)
	return defaults, err
}

// DefaultsSectionEdit applies defaults to the "haproxytech" defaults section, merging it
// over the built-in default values. Non-zero fields in defaults override or extend the
// defaults depending on mergeStrategy. Passing nil applies the defaults as-is.
func (c *clientNative) DefaultsSectionEdit(defaults *models.Defaults, mergeStrategy string) error {
	merged, err := deepCopyDefaults(c.defaultDefaults)
	if err != nil {
		return err
	}

	if defaults != nil {
		opts := []func(*mergo.Config){}
		switch mergeStrategy {
		case "override":
			opts = []func(*mergo.Config){mergo.WithOverride, mergo.WithOverrideEmptySlice}
		case "append":
			opts = []func(*mergo.Config){mergo.WithOverride, mergo.WithAppendSlice}
		}
		if err := mergo.Merge(&merged, defaults, opts...); err != nil {
			return err
		}
	}

	merged.Name = constants.DefaultsSectionName

	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return err
	}

	reload.Instance().SetReload("Defaults edited")

	_, existing, err := configuration.GetStructuredDefaultsSection(constants.DefaultsSectionName, c.activeTransaction)
	if err != nil || existing == nil {
		return configuration.CreateStructuredDefaultsSection(&merged, c.activeTransaction, 0)
	}
	return configuration.EditStructuredDefaultsSection(constants.DefaultsSectionName, &merged, c.activeTransaction, 0)
}

// deepCopyDefaults returns a deep copy of d via JSON round-trip.
func deepCopyDefaults(d models.Defaults) (models.Defaults, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return models.Defaults{}, err
	}
	var out models.Defaults
	if err := json.Unmarshal(b, &out); err != nil {
		return models.Defaults{}, err
	}
	return out, nil
}

// DeepCopyDefaultsPtr returns a deep copy of the given *models.Defaults. Returns nil, nil when original is nil.
func DeepCopyDefaultsPtr(original *models.Defaults) (*models.Defaults, error) {
	if original == nil {
		return nil, nil
	}
	copied, err := deepCopyDefaults(*original)
	if err != nil {
		return nil, err
	}
	return &copied, nil
}
