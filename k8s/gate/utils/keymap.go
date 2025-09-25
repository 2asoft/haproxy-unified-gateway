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

package utils // revive:disable:var-naming

import (
	"fmt"
	"strings" // KeyMap is a generic map with a custom key generator function if ey is complex type

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type KeyMap[KEY any, VALUE any] struct {
	data       map[string]VALUE
	computeKey func(KEY) string
}

func NewKeyMap[KEY any, VALUE any](keygen func(KEY) string) KeyMap[KEY, VALUE] {
	return KeyMap[KEY, VALUE]{
		data:       make(map[string]VALUE),
		computeKey: keygen,
	}
}

func (m *KeyMap[KEY, VALUE]) Get(key KEY) (VALUE, bool) {
	k := m.computeKey(key)
	v, ok := m.data[k]
	return v, ok
}

func (m *KeyMap[KEY, VALUE]) Exists(key KEY) bool {
	k := m.computeKey(key)
	_, ok := m.data[k]
	return ok
}

func (m *KeyMap[KEY, VALUE]) Set(key KEY, value VALUE) {
	k := m.computeKey(key)
	m.data[k] = value
}

func (m *KeyMap[KEY, VALUE]) Delete(key KEY) {
	k := m.computeKey(key)
	delete(m.data, k)
}

func (m *KeyMap[KEY, VALUE]) Iterate(callback func(string, VALUE) bool) {
	for k, v := range m.data {
		if !callback(k, v) {
			return
		}
	}
}

func (m *KeyMap[KEY, VALUE]) Len() int {
	return len(m.data)
}

func (m *KeyMap[KEY, VALUE]) Clear() {
	m.data = make(map[string]VALUE)
}

type copier[T any] interface {
	DeepCopy() T
}

// DeepCopy creates a deep copy of the KeyMap.
// It requires the VALUE type to have a DeepCopy() method.
func (m *KeyMap[KEY, VALUE]) DeepCopy() KeyMap[KEY, VALUE] {
	newMap := NewKeyMap[KEY, VALUE](m.computeKey)
	for k, v := range m.data {
		if c, ok := any(v).(copier[VALUE]); ok {
			newMap.data[k] = c.DeepCopy()
		}
	}
	return newMap
}

// ParentRefToKey converts a ParentReference to a unique string key.
// It handles nil pointers by using a consistent placeholder.
func ParentRefToKey(parentRef gatewayv1.ParentReference) string {
	var parts []string

	parts = append(parts, StrPtrToString(parentRef.Group))
	parts = append(parts, StrPtrToString(parentRef.Kind))
	parts = append(parts, StrPtrToString(parentRef.Namespace))
	parts = append(parts, string(parentRef.Name))
	parts = append(parts, StrPtrToString(parentRef.SectionName))
	// Port is not supported yet

	return strings.Join(parts, ":")
}

// KeyToParentRef converts a string key back to a gatewayv1.ParentReference.
// It is the inverse of ParentRefToKey.
func KeyToParentRef(key string) (gatewayv1.ParentReference, error) {
	parts := strings.Split(key, ":")
	if len(parts) != 5 {
		return gatewayv1.ParentReference{}, fmt.Errorf("invalid parent reference key: expected 5 parts, got %d", len(parts))
	}

	group := StringToPtr[gatewayv1.Group](parts[0])
	kind := StringToPtr[gatewayv1.Kind](parts[1])
	namespace := StringToPtr[gatewayv1.Namespace](parts[2])
	name := gatewayv1.ObjectName(parts[3])
	sectionName := StringToPtr[gatewayv1.SectionName](parts[4])

	return gatewayv1.ParentReference{
		Group:       group,
		Kind:        kind,
		Namespace:   namespace,
		Name:        name,
		SectionName: sectionName,
		// Port is not supported yet
	}, nil
}

// stringer is a constraint that allows any pointer to a type with an underlying string type.
type stringer interface {
	~string
}

func StrPtrToString[T stringer](p *T) string {
	if p == nil {
		return "_"
	}
	return string(*p)
}

// StringToPtr converts a string to a pointer of a string-based type.
// If the string is the placeholder "_", it returns nil.
func StringToPtr[T stringer](s string) *T {
	if s == "_" {
		return nil
	}
	return Ptr(T(s))
}
