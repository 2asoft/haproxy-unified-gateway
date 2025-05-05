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
	"reflect"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/events"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespacedNameFilterFunc is a function that returns true if the resource should be processed by the reconciler.
// If the function returns false, the reconciler will log the returned string.
type NamespacedNameFilterFunc func(nsname types.NamespacedName) (shouldProcess bool, msg string)

// ReconcilerConfig is the configuration for the reconciler.
type ReconcilerConfig struct {
	// Getter gets a resource from the k8s API.
	Getter Getter
	// ObjectType is the type of the resource that the reconciler will reconcile.
	ObjectType client.Object
	// EventCh is the channel where the reconciler will send events.
	EventCh chan<- any
	// NamespacedNameFilter filters resources the controller will process. Can be nil.
	NamespacedNameFilter NamespacedNameFilterFunc
	OnlyMetadata         bool
}

// Reconciler reconciles Kubernetes resources of a specific type.
// It implements the reconcile.Reconciler interface.
// A successful reconciliation of a resource has the two possible outcomes:
// (1) If the resource is deleted, the Implementation will send a DeleteEvent to the event channel.
// (2) If the resource is upserted (created or updated), the Implementation will send an UpsertEvent
// to the event channel.
type Reconciler struct {
	cfg ReconcilerConfig
}

var _ reconcile.Reconciler = &Reconciler{}

// NewReconciler creates a new reconciler.
func NewReconciler(cfg ReconcilerConfig) *Reconciler {
	return &Reconciler{
		cfg: cfg,
	}
}

func (r *Reconciler) mustCreateNewObject(objectType client.Object) client.Object {
	if r.cfg.OnlyMetadata {
		partialObj := &metav1.PartialObjectMetadata{}
		partialObj.SetGroupVersionKind(objectType.GetObjectKind().GroupVersionKind())

		return partialObj
	}

	t := reflect.TypeOf(objectType).Elem()
	obj, ok := reflect.New(t).Interface().(client.Object)
	if !ok {
		panic("failed to create a new object")
	}
	return obj
}

// Reconcile implements the reconcile.Reconciler Reconcile method.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// The controller runtime has already set the logger with the group, kind, namespace and name of the resource,
	logger.Info("Reconciling the resource")

	if r.cfg.NamespacedNameFilter != nil {
		if shouldProcess, msg := r.cfg.NamespacedNameFilter(req.NamespacedName); !shouldProcess {
			logger.Info(msg)
			return reconcile.Result{}, nil
		}
	}

	obj := r.mustCreateNewObject(r.cfg.ObjectType)

	if err := r.cfg.Getter.Get(ctx, req.NamespacedName, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "Failed to get the resource")
			return reconcile.Result{}, err
		}
		// The resource does not exist (was deleted).
		obj = nil
	}

	var e any
	var op string

	if obj == nil {
		e = &events.DeleteEvent{
			Type:           r.cfg.ObjectType,
			NamespacedName: req.NamespacedName,
		}
		op = "Deleted"
	} else {
		e = &events.UpsertEvent{
			Resource: obj,
		}
		op = "Upserted"
	}

	select {
	case <-ctx.Done():
		logger.Info("Did not process the resource because the context was canceled")
		return reconcile.Result{}, nil
	case r.cfg.EventCh <- e:
	}

	logger.Info(op + (" the resource"))

	return reconcile.Result{}, nil
}
