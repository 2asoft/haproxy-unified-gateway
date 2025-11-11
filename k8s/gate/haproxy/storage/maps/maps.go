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

package maps

import futils "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/fileutils"

type MapData struct {
	// Data contains the values
	// Only set if the map is not saved on disk
	// For HUG, Data will be empty as the map is saved on disk
	data map[string]string // TODO consider ordered map
	// Path is set only if the map was saved on disk (this is a gate configuration)
	// For HUG, maps are stored on disk
	Path futils.FilePath
	// Dynamic updates through runtime
	DynamicUpdates DynamicMapUpdates
	// ReloadNeeded is managed only if the gate library did try to run a runtime command (this is a gate configuration)
	// For HUG, the gate library will try to create/update/delete maps through runtime
	// By default, it is set to true
	// If the gate library did try to update the maps through runtime (create/update/delete), and a reload is not needed
	// as the runtime command did succeed, then the flag is set to false
	ReloadNeeded bool
}

// Data returns the map data
func (m *MapData) Data() map[string]string {
	return m.data
}

// SetData sets the map data
func (m *MapData) SetData(data map[string]string) {
	m.data = data
}

// AddData adds a key-value pair to the map
// If the key already exists, the value is updated
func (m *MapData) AddData(key, val string) {
	if m.data == nil {
		m.data = make(map[string]string)
	}
	_, exists := m.data[key]
	if exists {
		// key already exists, update value
		m.DynamicUpdates.Update[key] = val
	} else {
		// key does not exist, add new entry
		m.DynamicUpdates.Add[key] = val
	}
	// TODO question is, do we want to update the map here?
	// or just keep track of the changes to be applied later?
	// For now, we update the map directly
	// Reason is first update, we might have duplicate memory usage
	m.data[key] = val
}

// DeleteData deletes a key from the map
func (m *MapData) DeleteData(key string) {
	delete(m.data, key)
	m.DynamicUpdates.Delete = append(m.DynamicUpdates.Delete, key)
}

// DynamicMapUpdates contains the updates to be applied to the map on runtime (and to have diff)
type DynamicMapUpdates struct {
	// ToAdd contains the entries to add to the map
	Add    map[string]string
	Update map[string]string
	// ToDelete contains the entries to delete from the map
	Delete []string
}
