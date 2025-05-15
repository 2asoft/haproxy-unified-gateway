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
	"sync"
)

// CategoryFilterHandler filters log records by level and key-value category.
type CategoryFilterHandler struct {
	base              slog.Handler
	level             *slog.LevelVar
	allowedCategories map[string]bool
	categoryKey       string
	mu                sync.RWMutex
}

type CategoryFilterHandlerParams struct {
	Base              slog.Handler
	CategoryKey       string
	AllowedCategories []string
	InitialLevel      slog.Level
}

// NewCategoryFilterHandler wraps an existing handler and filters by level and category.
func NewCategoryFilterHandler(params CategoryFilterHandlerParams) *CategoryFilterHandler {
	allowed := make(map[string]bool, len(params.AllowedCategories))
	for _, cat := range params.AllowedCategories {
		allowed[cat] = true
	}
	alevel := &slog.LevelVar{}
	alevel.Set(params.InitialLevel)
	return &CategoryFilterHandler{
		base:              params.Base,
		level:             alevel,
		allowedCategories: allowed,
		categoryKey:       params.CategoryKey,
	}
}

func (h *CategoryFilterHandler) Enabled(_ context.Context, level slog.Level) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return level >= h.level.Level()
}

func (h *CategoryFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	allowed := true
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == h.categoryKey {
			if a.Value.Kind() == slog.KindString {
				strVal := a.Value.String()
				allowed = h.allowedCategories[strVal] || h.allowedCategories["all"]
			} else {
				allowed = false
			}
			return false // Stop scanning attributes early
		}
		// allowed = false // do not log if key categoryKey is not present
		return true
	})

	if !allowed {
		return nil
	}

	return h.base.Handle(ctx, r)
}

func (h *CategoryFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CategoryFilterHandler{
		base:              h.base.WithAttrs(attrs),
		level:             h.level,
		allowedCategories: h.allowedCategories,
		categoryKey:       h.categoryKey,
	}
}

func (h *CategoryFilterHandler) WithGroup(name string) slog.Handler {
	return &CategoryFilterHandler{
		base:              h.base.WithGroup(name),
		level:             h.level,
		allowedCategories: h.allowedCategories,
		categoryKey:       h.categoryKey,
	}
}

func (h *CategoryFilterHandler) SetLevel(level slog.Level) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.level.Set(level)
}

func (h *CategoryFilterHandler) GetLevel() slog.Level {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.level.Level()
}

func (h *CategoryFilterHandler) ReconcileAllowedCategories(categories []string) bool {
	newSet := make(map[string]bool, len(categories))
	for _, c := range categories {
		newSet[c] = true
	}

	h.mu.RLock()
	unchanged := mapsEqual(h.allowedCategories, newSet)
	h.mu.RUnlock()

	if unchanged {
		return false
	}

	h.mu.Lock()
	h.allowedCategories = newSet
	h.mu.Unlock()
	return true
}

func (h *CategoryFilterHandler) ReconcileLevel(level string) bool {
	var expectedLevel slog.Level
	switch level {
	case "Info":
		expectedLevel = slog.LevelInfo
	case "Warn":
		expectedLevel = slog.LevelWarn
	case "Error":
		expectedLevel = slog.LevelError
	case "Debug":
		expectedLevel = slog.LevelDebug
	}

	h.mu.RLock()
	unchanged := h.level.Level() == expectedLevel
	h.mu.RUnlock()

	if unchanged {
		return false
	}

	h.mu.Lock()
	h.level.Set(expectedLevel)
	h.mu.Unlock()
	return true
}

func mapsEqual(a, b map[string]bool) bool {
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
