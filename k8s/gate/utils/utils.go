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
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Ptr return pointer to a given value
func Ptr[V any](v V) *V {
	return &v
}

// ExtractGVK is a function that extracts the GroupVersionKind (GVK) of a client.object.
// It will log an error if the GKV cannot be extracted.
type ExtractGVK func(object client.Object) schema.GroupVersionKind

// NewExtractGKV creates a new MustExtractGVK function using the scheme.
func NewExtractGKV(scheme *runtime.Scheme, logger *slog.Logger) ExtractGVK {
	return func(obj client.Object) schema.GroupVersionKind {
		gvk, err := apiutil.GVKForObject(obj, scheme)
		if err != nil {
			// this should not happen
			logger.LogAttrs(context.Background(), slog.LevelError,
				fmt.Sprintf("could not extract GVK for object: %T", obj),
			)
		}

		return gvk
	}
}

type ObjectWithTimestamp interface {
	GetCreationTimestamp() metav1.Time
	GetName() string
}

func SortByCreationTimestamp[T ObjectWithTimestamp](objects []T) {
	sort.Slice(objects, func(i, j int) bool {
		a := objects[i]
		b := objects[j]
		aTime := a.GetCreationTimestamp()
		bTime := b.GetCreationTimestamp()
		return aTime.Time.Before(bTime.Time) ||
			(aTime.Time.Equal(bTime.Time) && a.GetName() < b.GetName())
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

func DeepCopyMap[K comparable, V any](src map[K]V) map[K]V {
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func NamespaceAsString(ns *gatewayv1.Namespace) string {
	if ns == nil {
		return ""
	}
	return string(*ns)
}

func PtrInt64(value int64) *int64 {
	return &value
}
