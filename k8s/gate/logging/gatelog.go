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
package logging

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LogCategoryKey = "category"
)

func LogAttrCategory(category LogCategory) slog.Attr {
	return slog.String("category", string(category))
}

func LogAttrResource(obj client.Object, gvk schema.GroupVersionKind) slog.Attr {
	return slog.Group("resource",
		slog.String("GVK", gvk.String()),
		LogAttrObjectKey(obj),
	)
}

func LogAttrObjectKey(obj client.Object) slog.Attr {
	return slog.String("objectKey", client.ObjectKeyFromObject(obj).String())
}

func LogAttrEventType(t string) slog.Attr {
	return slog.String("type", t)
}

func LogAttrBatch(id, length int) slog.Attr {
	return slog.Group("batch",
		slog.Int("id", id),
		slog.Int("length", length),
	)
}

func LogAttrDuration(duration time.Duration) slog.Attr {
	return slog.String("duration", duration.String())
}

func LogAttrError(err error) slog.Attr {
	return slog.String("error", err.Error())
}

func LogAttrLogLevel(level string) slog.Attr {
	return slog.String("logLevel", level)
}

func LogAttrLogCategories(catogories []string) slog.Attr {
	s := strings.Join(catogories, ", ")
	return slog.String("categories", s)
}

func LogAttrInstalledVersions(versions map[string]struct{}) slog.Attr {
	return slog.String("installedVersions", fmt.Sprintf("%v", versions))
}
