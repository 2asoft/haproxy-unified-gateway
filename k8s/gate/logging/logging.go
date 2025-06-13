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
	"context"
	"log/slog"
	"maps"
	"runtime"
	"sync"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
)

var (
	DefaultLogLevelPerCategory = map[v3.Category]slog.Level{
		LogCategoryK8s:    slog.LevelInfo,
		LogCategoryGate:   slog.LevelInfo,
		LogCategoryStatus: slog.LevelInfo,
	}
	DefaultLevel = slog.LevelInfo

	// logLevelPerCategory contains the seetings if a CR is deployed
	// if no CR is deployed, it's set to defaultLogLevelPerCategory
	logLevelPerCategory = make(map[v3.Category]slog.Level)
	// level is the level used when no category is set with this value if the conf CR
	level slog.Level
	mu    sync.RWMutex
)

// CategoryFilterHandler filters log records by level and key-value category.
type CategoryFilterHandler struct {
	base        slog.Handler
	categoryKey string
}

type CategoryFilterHandlerParams struct {
	Base                  slog.Handler
	DefaultCategoryLevels map[v3.Category]slog.Level
	CategoryKey           string
	DefaultLevel          slog.Level
}

// NewCategoryFilterHandler wraps an existing handler and filters by level and category.
func NewCategoryFilterHandler(params CategoryFilterHandlerParams) *CategoryFilterHandler {
	mu.Lock()
	defer mu.Unlock()
	DefaultLogLevelPerCategory = copyCategoryLevels(params.DefaultCategoryLevels)
	DefaultLevel = params.DefaultLevel
	logLevelPerCategory = copyCategoryLevels(params.DefaultCategoryLevels)

	return &CategoryFilterHandler{
		base:        params.Base,
		categoryKey: params.CategoryKey,
	}
}

func copyCategoryLevels(m map[v3.Category]slog.Level) map[v3.Category]slog.Level {
	cp := make(map[v3.Category]slog.Level, len(m))
	maps.Copy(cp, m)
	return cp
}

func GetLogSettings() (slog.Level, map[v3.Category]slog.Level) {
	mu.RLock()
	defer mu.RUnlock()
	return level, logLevelPerCategory
}

func (*CategoryFilterHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true // decision deferred to Handle
}

func (h *CategoryFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	mu.Lock()
	defer mu.Unlock()
	_, file, no, _ := runtime.Caller(3)
	r.AddAttrs(LogAttrFileSource(file, no))

	// Empty Category should happen only for k8s Logs
	category := LogCategoryK8s
	r.Attrs(func(a slog.Attr) bool {
		if a.Value.Kind() == slog.KindString && (a.Key == h.categoryKey || a.Key == "all") {
			category = v3.Category(a.Value.String())
			return false
		}
		return true
	})
	if category == LogCategoryK8s {
		// If no category is set, we use the default level
		r.AddAttrs(slog.String(h.categoryKey, string(category)))
	}

	catLevel, ok := logLevelPerCategory[category]
	if !ok {
		catLevel = level
	}

	if r.Level < catLevel {
		return nil
	}

	return h.base.Handle(ctx, r)
}

func (h *CategoryFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CategoryFilterHandler{
		base:        h.base.WithAttrs(attrs),
		categoryKey: h.categoryKey,
	}
}

func (h *CategoryFilterHandler) WithGroup(name string) slog.Handler {
	return &CategoryFilterHandler{
		base:        h.base.WithGroup(name),
		categoryKey: h.categoryKey,
	}
}

func (h *CategoryFilterHandler) ResetToDefaults() {
	_ = h.ReconcileLogSettings(DefaultLevel, DefaultLogLevelPerCategory)
}

func (*CategoryFilterHandler) ReconcileLogSettings(aLevel slog.Level, categories map[v3.Category]slog.Level) bool {
	mu.Lock()
	defer mu.Unlock()
	levelChanged := level != aLevel
	if levelChanged {
		level = aLevel
	}

	newSet := make(map[v3.Category]slog.Level, len(categories))
	maps.Copy(newSet, categories)
	changed := !mapsEqual(logLevelPerCategory, newSet)
	if changed {
		logLevelPerCategory = newSet
	}
	return levelChanged || changed
}

func LogLevelString2SlogLevel(level string) slog.Level {
	switch level {
	case "Info":
		return slog.LevelInfo
	case "Warn":
		return slog.LevelWarn
	case "Error":
		return slog.LevelError
	case "Debug":
		return slog.LevelDebug
	default:
		return slog.LevelInfo // Default to Info if unknown level
	}
}

func mapsEqual(a, b map[v3.Category]slog.Level) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
