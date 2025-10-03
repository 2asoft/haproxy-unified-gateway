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
package haproxy

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/metadata"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/templates"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/utils"

	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type BackendOwnerType string

const (
	BackendOwnerTypeHTTPRoute BackendOwnerType = "HTTPRoute"
)

type BackendReferencedBy struct {
	owners map[string]map[BackendOwnerType]map[string]int64 // map[backendName] -> map[ownerType] -> map[ownerName] -> generation
}

type BackendImpactedInCycle struct {
	Name         string
	HTTPRouteKey client.ObjectKey
	BackendRef   gatewayv1.HTTPBackendRef
}

type BackendsImpactedInCycle struct {
	Upserted map[string]map[client.ObjectKey]BackendImpactedInCycle // map[backendName] -> map [routeKey]
	Deleted  map[string]struct{}                                    // map[backendName]
	// For unreferenced backends, just update the metadata
	Unreferenced map[string]struct{} // map[backendName]
}

func NewBackendOwners() BackendReferencedBy {
	return BackendReferencedBy{owners: make(map[string]map[BackendOwnerType]map[string]int64)}
}

func (b *HaproxyConfMgrImpl) getBackendNameName(svcKey k8stypes.NamespacedName, svcPort int32, filterHash string) (string, error) {
	tmpl, err := template.New("backend").Parse(b.params.BackendNameTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse backend name template: %w", err)
	}

	data := templates.TemplateData{
		SERVICE_NAMESPACE: svcKey.Namespace,
		SERVICE_NAME:      svcKey.Name,
		SERVICE_PORT:      svcPort,
		FILTER_HASH:       filterHash,
		LINK_ID:           b.params.LinkID,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (b *HaproxyConfMgrImpl) processHTTPRoutes() error {
	var errs utils.Errors
	// Managed HTTPRoutes => Create / update/ delete backends
	for routeKey, route := range b.controllerStore.GateTree.HTTPRoutes {
		switch route.TreeStatus.Status {
		case store.StatusUnchanged:
			continue
		case store.StatusUpserted:
			err := b.onUpsertedHTTPRoute(routeKey, route)
			errs.Add(err)
		case store.StatusDeleted:
			err := b.onDeletedHTTPRoute(routeKey, route)
			errs.Add(err)
		}
	}

	// Cleanup Backends that are not referenced anymore
	if err := b.cleanupUnreferencedBackends(); err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to cleanup unreferenced backends",
			logging.LogAttrError(err),
		)
		errs.Add(err)
	}

	// Now we have the list of upserted + delete BE with the correct list of routes pointing to them
	if err := b.processBackendsModifiedInCycle(); err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to process backends modified in cycle",
			logging.LogAttrError(err),
		)
		errs.Add(err)
	}

	return errs.Result()
}

func (b *HaproxyConfMgrImpl) onUpsertedHTTPRoute(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute) error {
	switch route.Valid {
	case true:
		err := b.onValidHTTPRouteUpserted(routeKey, route)
		if err != nil {
			return err
		}
	case false:
		err := b.onInvalidHTTPRouteUpserted(routeKey, route)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *HaproxyConfMgrImpl) onValidHTTPRouteUpserted(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute) error {
	b.logHTTPRouteUpdate("upserted", routeKey)
	err := b.upsertBackends(routeKey, route)
	return err
}

func (b *HaproxyConfMgrImpl) onInvalidHTTPRouteUpserted(routeKey k8stypes.NamespacedName, _ *tree.HTTPRoute) error {
	b.logHTTPRouteUpdate("upserted-invalid", routeKey)
	impactedBackends := b.backendOwners.getBackendsReferencedByHTTPRoute(routeKey.String())
	for beName := range impactedBackends {
		b.backendOwners.removeHTTPRoute(beName, routeKey)
		b.addImpactedBackendDeleted(beName)
	}
	return nil
}

func (b *HaproxyConfMgrImpl) onDeletedHTTPRoute(routeKey k8stypes.NamespacedName, _ *tree.HTTPRoute) error {
	b.logHTTPRouteUpdate("deleted", routeKey)
	impactedBackends := b.backendOwners.getBackendsReferencedByHTTPRoute(routeKey.String())
	for beName := range impactedBackends {
		b.backendOwners.removeHTTPRoute(beName, routeKey)
		b.addImpactedBackendDeleted(beName)
	}
	return nil
}

func (b *HaproxyConfMgrImpl) logHTTPRouteUpdate(action string, key k8stypes.NamespacedName) {
	b.logger.LogAttrs(context.Background(), slog.LevelDebug, "Processing HTTPRoute ["+action+"]",
		logging.LogAttrKey(key),
	)
}

func (b *HaproxyConfMgrImpl) upsertBackends(routeKey k8stypes.NamespacedName, route *tree.HTTPRoute) error {
	var errs utils.Errors
	upsertedBackendsReferencedByRoute := make(map[string]struct{})
	for _, rule := range route.Rules {
		if rule.Valid {
			k8sRule := rule.K8sResource
			// Iterate now on each referenced Backend
			for _, backendRef := range k8sRule.BackendRefs {
				filterHash := getFilterHash(backendRef.Filters)
				var svcPort int32
				svcNsName := tree.ServiceNsNameKey(route.K8sResource, backendRef.BackendObjectReference)
				if backendRef.Port != nil {
					svcPort = int32(*backendRef.Port)
				}
				beName, err := b.getBackendNameName(svcNsName, svcPort, filterHash)
				if err != nil {
					b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to compute backendName",
						logging.LogAttrError(err))
					errs.Add(err)
					continue
				}

				// Check if backendRef is valid
				checkResult, ok := rule.CheckBackendRef.Get(backendRef.BackendObjectReference)
				if !ok {
					b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to check backendRef")
					continue
				}
				// If the specific backendCheck result is ok, add the BE
				if checkResult.Valid {
					err := b.backendOwners.addHTTPRoute(beName, routeKey, route.K8sResource)
					if err != nil {
						errs.Add(err)
						continue
					}
					b.addImpactedBackendUpserted(beName, routeKey, route.K8sResource, backendRef)
					upsertedBackendsReferencedByRoute[beName] = struct{}{}
				}
			}
		}
	}

	// Now cleanup the referenced Backends
	// For example:
	// Scenario:
	// Step1: Route1 references BE1, BE2, BE3
	// Step2: Update Route1 references BE1, BE2
	// Action: We have to remove BE3 from Backends referenced by Route1
	backendsReferencedByRoute := b.backendOwners.getBackendsReferencedByHTTPRoute(routeKey.String())
	unreferenced := utils.SetDifference(backendsReferencedByRoute, upsertedBackendsReferencedByRoute)
	for unreferencedBeName := range unreferenced {
		b.backendOwners.removeHTTPRoute(unreferencedBeName, routeKey)
		b.backendsImpactedInCycle.Deleted[unreferencedBeName] = struct{}{}
		b.backendsImpactedInCycle.Unreferenced[unreferencedBeName] = struct{}{}
	}
	return errs.Result()
}

func (b *HaproxyConfMgrImpl) addImpactedBackendUpserted(backendName string, routeKey client.ObjectKey,
	httpRoute *gatewayv1.HTTPRoute, httpBackendRef gatewayv1.HTTPBackendRef,
) {
	impactedBe := BackendImpactedInCycle{
		Name:         backendName,
		HTTPRouteKey: client.ObjectKeyFromObject(httpRoute),
		BackendRef:   httpBackendRef,
	}

	if _, ok := b.backendsImpactedInCycle.Upserted[backendName]; !ok {
		b.backendsImpactedInCycle.Upserted[backendName] = make(map[client.ObjectKey]BackendImpactedInCycle)
	}
	b.backendsImpactedInCycle.Upserted[backendName][routeKey] = impactedBe
}

func (b *HaproxyConfMgrImpl) addImpactedBackendDeleted(backendName string) {
	b.backendsImpactedInCycle.Deleted[backendName] = struct{}{}
}

func (bo *BackendReferencedBy) addHTTPRoute(backendName string, routeKey k8stypes.NamespacedName, route *gatewayv1.HTTPRoute) error {
	if route == nil {
		return errors.New("nil route")
	}
	routeName := routeKey.String()
	bo.add(backendName, BackendOwnerTypeHTTPRoute, routeName, route.Generation)
	return nil
}

func (bo *BackendReferencedBy) add(backendName string, ownerType BackendOwnerType, ownerName string, generation int64) {
	if _, ok := bo.owners[backendName]; !ok {
		bo.owners[backendName] = make(map[BackendOwnerType]map[string]int64)
	}
	if _, ok := bo.owners[backendName][ownerType]; !ok {
		bo.owners[backendName][ownerType] = make(map[string]int64)
	}
	bo.owners[backendName][ownerType][ownerName] = generation
}

func (bo *BackendReferencedBy) removeHTTPRoute(backendName string, routeKey k8stypes.NamespacedName) {
	routeName := routeKey.String()
	bo.remove(backendName, BackendOwnerTypeHTTPRoute, routeName)
}

func (bo *BackendReferencedBy) remove(backendName string, ownerType BackendOwnerType, ownerName string) {
	ownersForBackend, ok := bo.owners[backendName]
	if !ok {
		return
	}
	ownersForType, ok := ownersForBackend[ownerType]
	if !ok {
		return
	}
	delete(ownersForType, ownerName)
}

func (bo *BackendReferencedBy) getBackendsReferencedByHTTPRoute(ownerName string) map[string]struct{} { // map[beName] -> struct{}
	return bo.getBackendsReferencedBy(BackendOwnerTypeHTTPRoute, ownerName)
}

func (bo *BackendReferencedBy) getBackendsReferencedBy(ownerType BackendOwnerType, ownerName string) map[string]struct{} { // map[beName] -> struct{}
	backends := make(map[string]struct{})
	for backendName, ownersForBackend := range bo.owners {
		ownersForType, ok := ownersForBackend[ownerType]
		if !ok {
			continue
		}
		if _, ok := ownersForType[ownerName]; ok {
			backends[backendName] = struct{}{}
		}
	}
	return backends
}

func (b *HaproxyConfMgrImpl) cleanupUnreferencedBackends() error {
	return b.cleanupUnreferencedBackendsForHTTPRoutes(BackendOwnerTypeHTTPRoute)
}

func (b *HaproxyConfMgrImpl) cleanupUnreferencedBackendsForHTTPRoutes(ownerType BackendOwnerType) error {
	for beName, ownersForBackend := range b.backendOwners.owners {
		ownersForType, ok := ownersForBackend[ownerType]
		if !ok {
			continue
		}
		if len(ownersForType) == 0 {
			delete(ownersForBackend, ownerType)
		}
		if len(ownersForBackend) == 0 {
			delete(b.backendOwners.owners, beName)
			b.addImpactedBackendDeleted(beName)
		}
	}
	return nil
}

func (*HaproxyConfMgrImpl) newBackend(backendName string, md metadata.MetaData, _ gatewayv1.HTTPBackendRef) (*models.Backend, error) {
	return &models.Backend{
		BackendBase: models.BackendBase{
			Metadata: md,
			Name:     backendName,
			Mode:     "http",
		},
	}, nil
}

func getFilterHash(filters []gatewayv1.HTTPRouteFilter) string {
	if len(filters) == 0 {
		return "_"
	}
	// Marshal the filters to JSON
	jsonData, err := json.Marshal(filters)
	if err != nil {
		// This should ideally not happen with valid Gateway API objects.
		return "_"
	}

	// Compute MD5 hash
	hash := md5.Sum(jsonData)

	return hex.EncodeToString(hash[:])
}

func DeepCopyBackend(original *models.Backend) (*models.Backend, error) {
	if original == nil {
		return nil, nil
	}
	var copied models.Backend
	data, err := json.Marshal(original) // Serialize to JSON
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct
	return &copied, nil
}

func (b *HaproxyConfMgrImpl) processBackendsModifiedInCycle() error {
	var errs utils.Errors
	// UPSERTED
	for backendName := range b.backendsImpactedInCycle.Upserted {
		routesInfo := make(map[string]metadata.HTTPRouteMetadaInfo)
		var backendRef gatewayv1.HTTPBackendRef
		owners, ok := b.backendOwners.owners[backendName]
		if !ok {
			err := fmt.Errorf("could not find owner for backend %s", backendName)
			errs.Add(err)
			continue
		}
		ownersForHTTPRoute, ok := owners[BackendOwnerTypeHTTPRoute]
		if !ok {
			err := fmt.Errorf("could not find owners for type %s", BackendOwnerTypeHTTPRoute)
			errs.Add(err)
			continue
		}
		for routeKey, generation := range ownersForHTTPRoute {
			routesInfo[routeKey] = metadata.HTTPRouteMetadaInfo{
				OwnerType:  string(BackendOwnerTypeHTTPRoute),
				Generation: generation,
			}
		}
		beMd := b.metadataManager.BackendMetaData(routesInfo)

		be, err := b.newBackend(backendName, beMd, backendRef)
		if err != nil {
			errs.Add(err)
			continue
		}

		if err := b.configuration.upsertBackend(b.logger, be); err != nil {
			errs.Add(err)
			continue
		}
		if b.firstSync.flag {
			b.firstSync.backends[be.Name] = struct{}{}
		}
	}

	// UNREFERENCED
	for backendName := range b.backendsImpactedInCycle.Unreferenced {
		// Recompute Metadata and update BE
		routesInfo := make(map[string]metadata.HTTPRouteMetadaInfo)
		owners, ok := b.backendOwners.owners[backendName]
		if !ok {
			err := fmt.Errorf("could not find owner for backend %s", backendName)
			errs.Add(err)
			continue
		}
		ownersForHTTPRoute, ok := owners[BackendOwnerTypeHTTPRoute]
		if !ok {
			err := fmt.Errorf("could not find owners for type %s", BackendOwnerTypeHTTPRoute)
			errs.Add(err)
			continue
		}
		for routeKey, generation := range ownersForHTTPRoute {
			routesInfo[routeKey] = metadata.HTTPRouteMetadaInfo{
				OwnerType:  string(BackendOwnerTypeHTTPRoute),
				Generation: generation,
			}
		}
		beMd := b.metadataManager.BackendMetaData(routesInfo)
		if err := b.configuration.upsertBackendMetadata(b.logger, backendName, beMd); err != nil {
			errs.Add(err)
			continue
		}
		if b.firstSync.flag {
			b.firstSync.backends[backendName] = struct{}{}
		}
	}

	// DELETED
	for backendName := range b.backendsImpactedInCycle.Deleted {
		if err := b.configuration.deleteBackend(b.logger, backendName); err != nil {
			errs.Add(err)
			continue
		}
		if b.firstSync.flag {
			delete(b.firstSync.backends, backendName)
		}
	}

	return errs.Result()
}
