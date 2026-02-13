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
	"cmp"
	"fmt"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type ObjectWithTimestamp interface {
	GetCreationTimestamp() metav1.Time
	GetName() string
}

func SortByCreationTimestamp[T ObjectWithTimestamp](objects []T) {
	slices.SortFunc(objects, func(a, b T) int {
		aTime := a.GetCreationTimestamp()
		bTime := b.GetCreationTimestamp()
		if c := aTime.Time.Compare(bTime.Time); c != 0 {
			return c
		}
		return cmp.Compare(a.GetName(), b.GetName())
	})
}

func MapToSortedListByCreationTimestamp[T ObjectWithTimestamp](objects map[types.NamespacedName]T) []T {
	list := make([]T, 0, len(objects))
	for _, obj := range objects {
		list = append(list, obj)
	}
	SortByCreationTimestamp(list)
	return list
}

// ParseNamespacedName parses a "namespace/name" string into a NamespacedName.
func ParseNamespacedName(s string) (types.NamespacedName, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return types.NamespacedName{}, fmt.Errorf("invalid format: expected namespace/name, got %q", s)
	}
	return types.NamespacedName{Namespace: parts[0], Name: parts[1]}, nil
}

// Generic function to clear a map in place
func ClearMap[K comparable, V any](m map[K]V) {
	for k := range m {
		delete(m, k)
	}
}

func Keys[K comparable, V any](src map[K]V) map[K]struct{} {
	dst := make(map[K]struct{})
	for k := range src {
		dst[k] = struct{}{}
	}
	return dst
}

func NamespaceAsString(ns *gatewayv1.Namespace) string {
	if ns == nil {
		return ""
	}
	return string(*ns)
}

func ObjectKeyFromNamespacedName(s string) (client.ObjectKey, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		// Handle error: string is not in the correct format
		return client.ObjectKey{}, fmt.Errorf("invalid format: expected namespace/name, got %q", s)
	}
	return client.ObjectKey{
		Namespace: parts[0],
		Name:      parts[1],
	}, nil
}

func RouteGroupKindsToString(routesGK []gatewayv1.RouteGroupKind) string {
	kinds := make([]string, 0, len(routesGK))
	for _, kind := range routesGK {
		kinds = append(kinds, string(kind.Kind))
	}
	return fmt.Sprintf("[%s]", strings.Join(kinds, ", "))
}

// SetDifference finds the keys that are in mapA but not in mapB.
// It works for any map with a comparable key type K and any value type V.
func SetDifference[K comparable, V any](mapA, mapB map[K]V) map[K]struct{} {
	diff := make(map[K]struct{})
	for key := range mapA {
		if _, exists := mapB[key]; !exists {
			diff[key] = struct{}{}
		}
	}
	return diff
}

func SetIntersection[K comparable, V any](mapA, mapB map[K]V) map[K]struct{} {
	intersection := make(map[K]struct{})
	for key := range mapA {
		if _, exists := mapB[key]; exists {
			intersection[key] = struct{}{}
		}
	}
	return intersection
}

// ComparePointers compares two pointers to any ordered type.
// It establishes a consistent sort order where nil values come before non-nil values.
//
// It returns:
//   - -1 if a < b
//   - 0 if a == b (or both are nil)
//   - +1 if a > b
func ComparePointers[T cmp.Ordered](a, b *T) int {
	if a == nil && b != nil {
		return -1 // nil comes before non-nil
	}
	if a != nil && b == nil {
		return 1 // non-nil comes after nil
	}
	if a != nil && b != nil {
		// Both are non-nil, compare their values.
		return cmp.Compare(*a, *b)
	}
	// Both are nil, so they are equal.
	return 0
}

// PointerDefaultValueIfNil dereferences a pointer and returns its value.
// If the pointer is nil, the zero value of T is returned instead.
//
// Example:
//
//	x := 42
//	v := PointerDefaultValueIfNil(&x)  // returns 42
//	v := PointerDefaultValueIfNil(nil) // returns 0 (zero value of int)
func PointerDefaultValueIfNil[T any](arg *T) T {
	if arg == nil {
		var a T
		return a
	}
	return *arg
}

func GetNamespacedName(name, namespace, defaultNamespace string) types.NamespacedName {
	if namespace == "" {
		namespace = defaultNamespace
	}
	return types.NamespacedName{Name: name, Namespace: namespace}
}

// GetHostnamesForRouteWithListener returns the hostnames that match a listener and a route.
// The rules are based on the Gateway API specification.
// If the listener hostname is not set, it matches any route hostname.
// If the listener hostname is a wildcard, it checks if it matches any route hostname.
// If the route hostnames are empty, the listener hostname matches the route hostnames.
func GetHostnamesForRouteWithListener(listenerHostname *string, routeHostnames []string) []string {
	if len(routeHostnames) == 0 && PointerDefaultValueIfNil(listenerHostname) == "" {
		return []string{""}
	}

	if PointerDefaultValueIfNil(listenerHostname) == "" {
		// no restriction from listeners, all hostnames from route are allowed
		return routeHostnames
	}

	if len(routeHostnames) == 0 {
		// route hostnames are empty, the listener hostname becomes the restriction
		return []string{*listenerHostname}
	}

	// to avoid duplicates
	matched := map[string]struct{}{}
	for _, routeHostname := range routeHostnames {
		// If the listener hostname is a wildcard, check if it matches any route hostname.
		if hostnamesMatchingRouteAndListener(routeHostname, *listenerHostname) {
			matched[*listenerHostname] = struct{}{}
		} else if hostnamesMatchingRouteAndListener(*listenerHostname, routeHostname) {
			// If the route hostname is a wildcard, check if it matches any listener hostname.
			matched[routeHostname] = struct{}{}
		}
	}
	result := make([]string, len(matched))
	i := 0
	for k := range matched {
		result[i] = k
		i++
	}
	slices.Sort(result)
	return result
}

func hostnamesMatchingRouteAndListener(pattern, hostname string) bool {
	// Exact match if no wildcard
	if !strings.HasPrefix(pattern, "*.") {
		return pattern == hostname
	}

	// pattern is of the form "*.example.com"
	return strings.HasSuffix(hostname, pattern[1:])
}

func ConvertSliceWithFunc[U, V any](arg []U, f func(U) V) []V {
	result := make([]V, len(arg))
	for i, v := range arg {
		result[i] = f(v)
	}
	return result
}

func ConvertV1Alpha2HostnameToString(hostname v1alpha2.Hostname) string {
	return string(hostname)
}
