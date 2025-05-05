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

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"

	ctlr "sigs.k8s.io/controller-runtime"
	ctlr_builder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// addIndexFieldTimeout is the timeout used for adding an Index Field to a cache.
	addIndexFieldTimeout = 2 * time.Minute
)

type recConfig struct {
	namespacedNameFilter NamespacedNameFilterFunc
	k8sPredicate         predicate.Predicate
	fieldIndices         index.FieldIndices
	newReconciler        NewReconcilerFunc
	onlyMetadata         bool
}

type NewReconcilerFunc func(cfg ReconcilerConfig) *Reconciler

// Option defines configuration options for registering a controller.
type Option func(*recConfig)

// WithNamespacedNameFilter enables filtering of objects by NamespacedName by the controller.
func WithNamespacedNameFilter(filter NamespacedNameFilterFunc) Option {
	return func(cfg *recConfig) {
		cfg.namespacedNameFilter = filter
	}
}

// WithK8sPredicate enables filtering of events before they are sent to the controller.
func WithK8sPredicate(p predicate.Predicate) Option {
	return func(cfg *recConfig) {
		cfg.k8sPredicate = p
	}
}

// WithFieldIndices adds indices to the FieldIndexer of the manager.
func WithFieldIndices(fieldIndices index.FieldIndices) Option {
	return func(cfg *recConfig) {
		cfg.fieldIndices = fieldIndices
	}
}

// WithNewReconciler allows us to mock reconciler creation in the unit tests.
func WithNewReconciler(newReconciler NewReconcilerFunc) Option {
	return func(cfg *recConfig) {
		cfg.newReconciler = newReconciler
	}
}

// WithOnlyMetadata tells the controller to only cache metadata, and to watch the API server in metadata-only form.
func WithOnlyMetadata() Option {
	return func(cfg *recConfig) {
		cfg.onlyMetadata = true
	}
}

func defaultConfig() recConfig {
	return recConfig{
		newReconciler: NewReconciler,
	}
}

// Register registers a new controller for the object type in the manager and configure it with the provided options.
// If the options include WithFieldIndices, it will add the specified indices to FieldIndexer of the manager.
// The registered controller will send events to the provided channel.
func Register(
	ctx context.Context,
	objectType client.Object,
	name string,
	mgr manager.Manager,
	eventCh chan<- any,
	options ...Option,
) error {
	cfg := defaultConfig()

	for _, opt := range options {
		opt(&cfg)
	}

	for field, indexerFunc := range cfg.fieldIndices {
		if err := addIndex(
			ctx,
			mgr.GetFieldIndexer(),
			objectType,
			field,
			indexerFunc,
		); err != nil {
			return err
		}
	}

	var forOpts []ctlr_builder.ForOption
	// .For
	// // This is the equivalent of calling
	// Watches(source.Kind(cache, &Type{}, &handler.EnqueueRequestForObject{})).
	// It would be possible to enqueue more dependent object reconcilitions.
	if cfg.onlyMetadata {
		if objectType.GetObjectKind().GroupVersionKind().Empty() {
			panic("the object must have its GVK set")
		}
		forOpts = append(forOpts, ctlr_builder.OnlyMetadata)
	}
	builder := ctlr.NewControllerManagedBy(mgr).
		Named(name).
		For(objectType, forOpts...)

	if cfg.k8sPredicate != nil {
		builder = builder.WithEventFilter(cfg.k8sPredicate)
	}

	reconcileConfig := ReconcilerConfig{
		Getter:               mgr.GetClient(),
		ObjectType:           objectType,
		EventCh:              eventCh,
		NamespacedNameFilter: cfg.namespacedNameFilter,
		OnlyMetadata:         cfg.onlyMetadata,
	}

	if err := builder.Complete(cfg.newReconciler(reconcileConfig)); err != nil {
		return fmt.Errorf("cannot build a controller for %T: %w", objectType, err)
	}

	return nil
}

func addIndex(
	ctx context.Context,
	indexer client.FieldIndexer,
	objectType client.Object,
	field string,
	indexerFunc client.IndexerFunc,
) error {
	c, cancel := context.WithTimeout(ctx, addIndexFieldTimeout)
	defer cancel()

	if err := indexer.IndexField(c, objectType, field, indexerFunc); err != nil {
		return fmt.Errorf("failed to add index for %T for field %s: %w", objectType, field, err)
	}

	return nil
}
