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

	utilsk8s "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils-k8s"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// enqueueGatewayClassForHugGate returns a handler.EventHandler that enqueues all GatewayClasses
// related to an observed HugGate.
// The relationship is built via the `spec.parametersRef` field in the GatewayClass.
func enqueueGatewayClassForHugGate(ctrlclient client.Client, _ utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		gwcList := &gatewayv1.GatewayClassList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclient.List(ctx, gwcList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, gwc := range gwcList.Items {
			if paramsRef, ok := getGatewayClassParamsRefKey(gwc); ok {
				if paramsRef.Name == o.GetName() && paramsRef.Namespace == o.GetNamespace() {
					requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
						Namespace: gwc.GetNamespace(),
						Name:      gwc.GetName(),
					}})
				}
			}
		}
		return requests
	}
}

func getGatewayClassParamsRefKey(gwc gatewayv1.GatewayClass) (types.NamespacedName, bool) {
	paramsRef := gwc.Spec.ParametersRef
	if paramsRef == nil {
		return types.NamespacedName{}, false
	}
	return client.ObjectKey{Namespace: utils.NamespaceAsString(paramsRef.Namespace), Name: paramsRef.Name}, true
}

// enqueueGatewayForHugGate returns a handler.EventHandler that enqueues all Gateway:
// Direct:
// - related to an observed HugGate.
// - The relationship is built via the `spec.parametersRef` field in the GatewayClass.
// Indirect:
// - related to the referenced GatewayClass that references this HugGate
func enqueueGatewayForHugGate(ctrlclint client.Client, _ utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// Gateways
		gwList := &gatewayv1.GatewayList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclint.List(ctx, gwList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		// GatewayClasses
		gwcList := &gatewayv1.GatewayClassList{}
		if err := ctrlclint.List(ctx, gwcList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, gw := range gwList.Items {
			// 1. Direct HugGate reference
			if paramsRef, ok := getGatewayParamsRefKey(gw); ok {
				if paramsRef.Name == o.GetName() && paramsRef.Namespace == o.GetNamespace() {
					requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
						Namespace: gw.GetNamespace(),
						Name:      gw.GetName(),
					}})
				}
			}
			// 2. Gateway references a GatewayClass that refenrences this HugGate
			gcName := string(gw.Spec.GatewayClassName)
			for _, gwc := range gwcList.Items {
				if gcName == gwc.Name {
					if paramsRef, ok := getGatewayClassParamsRefKey(gwc); ok {
						if paramsRef.Name == o.GetName() && paramsRef.Namespace == o.GetNamespace() {
							requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
								Namespace: gw.GetNamespace(),
								Name:      gw.GetName(),
							}},
							)
						}
					}
				}
			}
		}
		return requests
	}
}

func getGatewayParamsRefKey(gw gatewayv1.Gateway) (types.NamespacedName, bool) {
	if gw.Spec.Infrastructure == nil {
		return types.NamespacedName{}, false
	}
	if gw.Spec.Infrastructure.ParametersRef == nil {
		return types.NamespacedName{}, false
	}
	paramsRef := gw.Spec.Infrastructure.ParametersRef
	if paramsRef == nil {
		return types.NamespacedName{}, false
	}
	return client.ObjectKey{Namespace: gw.Namespace, Name: paramsRef.Name}, true
}

// enqueueGatewayForGatewayClass returns a handler.EventHandler that enqueues all Gateways
// related to an observed GatewayClass.
func enqueueGatewayForGatewayClass(ctrlclient client.Client, _ utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// Gateways
		gwList := &gatewayv1.GatewayList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclient.List(ctx, gwList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, gw := range gwList.Items {
			gwcName := string(gw.Spec.GatewayClassName)
			if gwcName == o.GetName() {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Namespace: gw.GetNamespace(),
					Name:      gw.GetName(),
				}})
			}
		}
		return requests
	}
}

// enqueueGatewayForSecret returns a handler.EventHandler that enqueues all Gateways
// related to an observed Secret.
func enqueueGatewayForSecret(ctrlclient client.Client, _ utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// Gateways
		gwList := &gatewayv1.GatewayList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclient.List(ctx, gwList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, gw := range gwList.Items {
			for _, listener := range gw.Spec.Listeners {
				if listener.TLS == nil {
					continue
				}
				for _, certRef := range listener.TLS.CertificateRefs {
					// We only accept v1.Secret
					if !utilsk8s.IsSecretGroupKindSupported(certRef) {
						continue
					}
					secretNsName := utils.GetNamespacedName(string(certRef.Name),
						string(utils.PointerDefaultValueIfNil(certRef.Namespace)),
						gw.GetNamespace())

					if secretNsName.Name == o.GetName() && secretNsName.Namespace == o.GetNamespace() {
						requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
							Namespace: gw.GetNamespace(),
							Name:      gw.GetName(),
						}})
					}
				}
			}
		}
		return requests
	}
}

// enqueueHTTPRouteForGateway returns a handler.EventHandler that enqueues all HTTPRoutes
// related to an observed Gateway.
func enqueueHTTPRouteForGateway(ctrlclient client.Client, extractGVK utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// HTTPRoutes
		routeList := &gatewayv1.HTTPRouteList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclient.List(ctx, routeList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, route := range routeList.Items {
			for _, parentRef := range route.Spec.ParentRefs {
				// We only accept v1.Gateway
				if !utilsk8s.IsParentRefGroupKindSupported(parentRef, extractGVK) {
					continue
				}
				gwNsName := utils.GetNamespacedName(string(parentRef.Name),
					string(utils.PointerDefaultValueIfNil(parentRef.Namespace)),
					route.GetNamespace())

				if gwNsName.Name == o.GetName() && gwNsName.Namespace == o.GetNamespace() {
					requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
						Namespace: route.GetNamespace(),
						Name:      route.GetName(),
					}})
				}
			}
		}

		return requests
	}
}

// enqueueHTTPRouteForService returns a handler.EventHandler that enqueues all HTTPRoutes
// related to an observed Service.
func enqueueHTTPRouteForService(ctrlclient client.Client, extractGVK utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// HTTPRoutes
		routeList := &gatewayv1.HTTPRouteList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclient.List(ctx, routeList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, route := range routeList.Items {
			for _, rule := range route.Spec.Rules {
				for _, backendRef := range rule.BackendRefs {
					// We only accept v1.Service
					if !utilsk8s.IsBackendRefGroupKindSupported(backendRef.BackendObjectReference, extractGVK) {
						continue
					}
					serviceNsName := utils.GetNamespacedName(
						string(backendRef.Name),
						string(utils.PointerDefaultValueIfNil(backendRef.Namespace)),
						route.GetNamespace())

					if serviceNsName.Name == o.GetName() && serviceNsName.Namespace == o.GetNamespace() {
						requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
							Namespace: route.GetNamespace(),
							Name:      route.GetName(),
						}})
					}
				}
			}
		}

		return requests
	}
}

// enqueueHTTPRouteForBackendCR returns a handler.EventHandler that enqueues all HTTPRoutes
// related to an observed Backend.
func enqueueHTTPRouteForBackendCR(ctrlclient client.Client, extractGVK utilsk8s.ExtractGVK) handler.MapFunc {
	return func(ctx context.Context, o client.Object) []reconcile.Request {
		var requests []reconcile.Request

		// HTTPRoutes
		routeList := &gatewayv1.HTTPRouteList{}

		listOpts := &client.ListOptions{}
		if err := ctrlclient.List(ctx, routeList, listOpts); err != nil {
			return []reconcile.Request{}
		}

		for _, route := range routeList.Items {
			for _, rule := range route.Spec.Rules {
				// BackendRef Filters
				for _, backendRef := range rule.BackendRefs {
					for _, filter := range backendRef.Filters {
						if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef {
							continue
						}
						// We only accept v3.Backend
						if !utilsk8s.IsFilterExtensionRefKindSupported(filter.ExtensionRef, extractGVK) {
							continue
						}
						nsName := types.NamespacedName{
							Namespace: route.Namespace,
							Name:      string(filter.ExtensionRef.Name),
						}
						if nsName.Name == o.GetName() && nsName.Namespace == o.GetNamespace() {
							requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
								Namespace: route.GetNamespace(),
								Name:      route.GetName(),
							}})
						}
					}
				}
			}
		}
		return requests
	}
}
