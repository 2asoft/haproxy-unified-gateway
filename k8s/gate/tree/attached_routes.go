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
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
)

// AttachedRoutes is a map from a route's NamespacedName to a placeholder struct.
type AttachedRoutes map[types.NamespacedName]struct{} // map[routeKey]

// MarshalJSON implements the json.Marshaler interface for AttachedRoutes.
func (m AttachedRoutes) MarshalJSON() ([]byte, error) {
	// Create a temporary map with string keys.
	tempMap := make(map[string]struct{})
	for k := range m {
		// Use "namespace/name" as the string key.
		keyStr := k.String()
		tempMap[keyStr] = struct{}{}
	}
	// Marshal the temporary map.
	return json.Marshal(tempMap)
}

// UnmarshalJSON implements the json.Unmarshaler interface for AttachedRoutes.
func (m *AttachedRoutes) UnmarshalJSON(data []byte) error {
	// Unmarshal the JSON into a temporary map with string keys.
	tempMap := make(map[string]struct{})
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return err
	}

	// Initialize the receiver map.
	*m = make(AttachedRoutes)

	// Iterate and reconstruct the NamespacedName keys.
	for kStr := range tempMap {
		parts := strings.SplitN(kStr, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key format: %s, expected namespace/name", kStr)
		}

		key := types.NamespacedName{
			Namespace: parts[0],
			Name:      parts[1],
		}
		(*m)[key] = struct{}{}
	}

	return nil
}
