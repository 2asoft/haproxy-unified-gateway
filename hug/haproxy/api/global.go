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
package api

import (
	"encoding/json"

	"github.com/imdario/mergo"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/hug/reload"
)

// GlobalGet returns the current global configuration from HAProxy.
// Returns an empty Global if none is set.
func (c *clientNative) GlobalGet() (models.Global, error) {
	config, err := c.nativeAPI.Configuration()
	if err != nil {
		return models.Global{}, err
	}
	_, global, err := config.GetStructuredGlobalConfiguration(c.activeTransaction)
	if err != nil {
		return models.Global{}, err
	}
	if global == nil {
		return models.Global{}, nil
	}
	return *global, nil
}

// GlobalEdit applies global to the HAProxy configuration, merging it over the
// default global (which carries the runtime API address and PID file derived
// from startup parameters). Non-zero fields in global override the defaults;
// omitted fields keep their default values. Passing nil applies the defaults as-is.
func (c *clientNative) GlobalEdit(global *models.Global, mergeStrategy string) error {
	// Start from a deep copy of the pre-built default so runtime_api.address
	// and pidfile are always set from params, even when the caller omits them.
	// A deep copy is required because models.Global contains pointer/slice fields
	// that mergo would otherwise mutate in place, permanently drifting defaultGlobal.
	merged, err := deepCopyGlobal(c.defaultGlobal)
	if err != nil {
		return err
	}

	if global != nil {
		opts := []func(*mergo.Config){}
		switch mergeStrategy {
		case "override":
			opts = []func(*mergo.Config){mergo.WithOverride, mergo.WithOverrideEmptySlice}
		case "append":
			opts = []func(*mergo.Config){mergo.WithOverride, mergo.WithAppendSlice}
		}
		if err := mergo.Merge(&merged, global, opts...); err != nil {
			return err
		}
	}

	applyMandatoryGlobal(&merged, c.mandatoryGlobal)

	config, err := c.nativeAPI.Configuration()
	if err != nil {
		return err
	}
	reload.Instance().SetReload("Global edited")
	return config.PushStructuredGlobalConfiguration(&merged, c.activeTransaction, 0)
}

// applyMandatoryGlobal enforces mandatory settings over merged:
//   - scalar fields: mandatory overrides (non-zero values win)
//   - RuntimeAPIs: mandatory entry is guaranteed at position 0, no duplicates (by address)
//   - LogTargetList: mandatory entries appended when not already present (by all fields)
func applyMandatoryGlobal(merged *models.Global, mand models.Global) {
	mandRuntimeAPIs := mand.RuntimeAPIs

	// Nil out lists so mergo only touches scalar fields.
	mand.RuntimeAPIs = nil
	_ = mergo.Merge(merged, mand, mergo.WithOverride)

	merged.RuntimeAPIs = mandatoryFirstRuntimeAPIs(merged.RuntimeAPIs, mandRuntimeAPIs)
}

// deepCopyGlobal returns a deep copy of g via JSON round-trip.
// This is necessary because models.Global contains pointer/slice fields that
// would otherwise be shared between the copy and the original.
func deepCopyGlobal(g models.Global) (models.Global, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return models.Global{}, err
	}
	var out models.Global
	if err := json.Unmarshal(b, &out); err != nil {
		return models.Global{}, err
	}
	return out, nil
}

// mandatoryFirstRuntimeAPIs places mandatory entries at the front of the list,
// then appends any existing entries whose address is not already covered.
func mandatoryFirstRuntimeAPIs(current []*models.RuntimeAPI, mandatory []*models.RuntimeAPI) []*models.RuntimeAPI {
	mandAddrs := make(map[string]struct{}, len(mandatory))
	for _, r := range mandatory {
		if r != nil && r.Address != nil {
			mandAddrs[*r.Address] = struct{}{}
		}
	}
	result := make([]*models.RuntimeAPI, 0, len(mandatory)+len(current))
	result = append(result, mandatory...)
	for _, r := range current {
		if r == nil || r.Address == nil {
			continue
		}
		if _, dup := mandAddrs[*r.Address]; !dup {
			result = append(result, r)
		}
	}
	return result
}
