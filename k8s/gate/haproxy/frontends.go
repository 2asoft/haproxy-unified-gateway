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
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/templates"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type FrontendsOwnedbyGateway struct {
	current map[client.ObjectKey]map[string]struct{} // map[gwKey] -> map[frontendName]struct{}
	updated map[client.ObjectKey]map[string]struct{} // map[gwKey] -> map[frontendName]struct{}
}

func (b *HaproxyConfMgrImpl) getFrontendName(gwKey k8stypes.NamespacedName, listener gatewayv1.Listener) (string, error) {
	tmpl, err := template.New("frontend").Parse(b.params.FrontendNameTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontend name template: %w", err)
	}

	data := templates.TemplateData{
		GATEWAY_NAMESPACE: gwKey.Namespace,
		GATEWAY_NAME:      gwKey.Name,
		LISTENER_NAME:     string(listener.Name),
		LINK_ID:           b.params.LinkID,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (b *HaproxyConfMgrImpl) processGateways() error {
	// Managed Gateways => Create / update/ delete frontends
	for gwKey, gateway := range b.controllerStore.GateTree.Gateways {
		switch gateway.TreeStatus.Status {
		case store.StatusUnchanged:
			continue
		case store.StatusUpserted:
			err := b.onUpsertedGateway(gwKey, gateway)
			if err != nil {
				return err
			}
		case store.StatusDeleted:
			err := b.onDeletedGateway(gwKey, gateway)
			if err != nil {
				return err
			}
		}
	}

	// Unmanaged Gateways => Delete frontends
	for gwKey, gateway := range b.controllerStore.UnmanagedGateTree.Gateways {
		err := b.onUnmanagedGateway(gwKey, gateway)
		if err != nil {
			return err
		}
	}

	// Cleanup frontends for gateways that were updated
	if err := b.cleanupFrontendsForGateways(); err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to cleanup frontends for gateways",
			logging.LogAttrError(err),
		)
	}

	// Finalize frontends by gateway
	b.finalizeFrontendsByGateway()
	return nil
}

func (b *HaproxyConfMgrImpl) onUpsertedGateway(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	switch gw.Valid {
	case true:
		err := b.onValidGatewayUpserted(gwKey, gw)
		if err != nil {
			return err
		}
	case false:
		err := b.onInvalidGatewayUpserted(gwKey, gw)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *HaproxyConfMgrImpl) onValidGatewayUpserted(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	b.logGatewayUpdate("upserted", gwKey)
	err := b.upsertFrontends(gwKey, gw)
	if err != nil {
		return err
	}
	return nil
}

func (b *HaproxyConfMgrImpl) onInvalidGatewayUpserted(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	b.logGatewayUpdate("upserted-invalid", gwKey)
	err := b.deleteFrontendForAllListeners(gwKey, gw)
	if err != nil {
		return err
	}
	return nil
}

func (b *HaproxyConfMgrImpl) onDeletedGateway(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	b.logGatewayUpdate("deleted", gwKey)
	err := b.deleteFrontendForAllListeners(gwKey, gw)
	if err != nil {
		return err
	}
	return nil
}

func (b *HaproxyConfMgrImpl) onUnmanagedGateway(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	b.logGatewayUpdate("unmanaged", gwKey)
	err := b.deleteFrontendForAllListeners(gwKey, gw)
	if err != nil {
		return err
	}
	return nil
}

func (b *HaproxyConfMgrImpl) upsertFrontends(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	for _, listener := range gw.Listeners {
		if !listener.Valid {
			return b.deleteFrontendForListener(gwKey, listener.K8sResource)
		}

		newFe, err := b.newFrontend(gwKey, gw, listener)
		if err != nil {
			return err
		}
		if b.firstSync {
			b.frontendsContainedInFirstSync[newFe.Name] = struct{}{}
		}
		if err := b.configuration.upsertFrontend(b.logger, newFe); err != nil {
			b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to upsert frontend",
				logging.LogAttrFrontendName(newFe.Name),
				logging.LogAttrError(err))
			continue
		}
		b.frontendsOwnedbyGateway.AddToUpdated(gwKey, newFe.Name)
	}

	return nil
}

func (b *HaproxyConfMgrImpl) cleanupFrontendsForGateways() error {
	// For each updated gateway, check if the frontend is still present
	for gwKey := range b.frontendsOwnedbyGateway.updated {
		for frontendName := range b.frontendsOwnedbyGateway.current[gwKey] {
			b.logger.LogAttrs(context.Background(), slog.LevelDebug, "Cleaning up frontend for gateway",
				logging.LogAttrKey(gwKey),
				slog.Any("current frontends", b.frontendsOwnedbyGateway.current[gwKey]))
			if _, ok := b.frontendsOwnedbyGateway.updated[gwKey][frontendName]; !ok {
				if err := b.deleteFrontend(gwKey, frontendName); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (b *HaproxyConfMgrImpl) finalizeFrontendsByGateway() {
	// For each updated gateway, check if the frontend is still present
	for gwKey := range b.frontendsOwnedbyGateway.updated {
		b.frontendsOwnedbyGateway.current[gwKey] = b.frontendsOwnedbyGateway.updated[gwKey]
		delete(b.frontendsOwnedbyGateway.updated, gwKey)
	}
}

func (b *HaproxyConfMgrImpl) newFrontend(gwKey k8stypes.NamespacedName, treeGw *tree.Gateway, treeListener *tree.Listener) (*models.Frontend, error) {
	listener := treeListener.K8sResource

	// Create a frontend for each listener
	feName, err := b.getFrontendName(gwKey, listener)
	if err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError,
			"Failed to get frontend name",
			slog.String("frontendNameTemplate", b.params.FrontendNameTemplate),
			logging.LogAttrKey(gwKey))
		return nil, fmt.Errorf("failed to get frontend name: %w", err)
	}

	md := b.metadataManager.FrontendMetaData(treeGw)
	fe := &models.Frontend{
		FrontendBase: models.FrontendBase{
			Name:     feName,
			From:     b.params.DefaultsSectionName,
			Metadata: md,
			// TODO: remove this, only temporary for test
			DefaultBackend: "be_tmp_test",
			Mode: func() string {
				if listener.Protocol == gatewayv1.HTTPProtocolType || listener.Protocol == gatewayv1.HTTPSProtocolType {
					return "http"
				}
				if listener.Protocol == gatewayv1.TCPProtocolType {
					return "tcp"
				}
				return ""
			}(),
		},
	}
	// Set other frontend properties based on the listener
	port := int64(listener.Port)
	if !b.params.DisableIPv4 {
		bind := models.Bind{
			Port: &port,
			Address: func() string {
				if b.params.IPv4BindAddress != "" {
					return b.params.IPv4BindAddress
				}
				return "0.0.0.0"
			}(),
			BindParams: b.bindParams(feName, "v4", treeGw, treeListener),
		}
		if fe.Binds == nil {
			fe.Binds = make(map[string]models.Bind)
		}
		fe.Binds[bind.Name] = bind
	}
	if !b.params.DisableIPv6 {
		bind := models.Bind{
			Port: &port,
			Address: func() string {
				if b.params.IPv6BindAddress != "" {
					return b.params.IPv6BindAddress
				}
				return "::"
			}(),
			BindParams: b.bindParams(feName, "v6", treeGw, treeListener),
		}
		if fe.Binds == nil {
			fe.Binds = make(map[string]models.Bind)
		}
		fe.Binds[bind.Name] = bind
	}
	return fe, nil
}

func (b *HaproxyConfMgrImpl) deleteFrontendForAllListeners(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	// K8s resource might be in:
	k8sGateway := gw.GetK8sResource()
	if k8sGateway == nil {
		return fmt.Errorf("no K8s resource found for gateway %s", gwKey)
	}

	for _, listener := range k8sGateway.Spec.Listeners {
		if err := b.deleteFrontendForListener(gwKey, listener); err != nil {
			continue
		}
	}
	return nil
}

func (b *HaproxyConfMgrImpl) deleteFrontendForListener(gwKey k8stypes.NamespacedName, listener gatewayv1.Listener) error {
	// Frontend for each listener
	feName, err := b.getFrontendName(gwKey, listener)
	if err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to get frontend name",
			slog.String("frontendNameTemplate", b.params.FrontendNameTemplate),
			logging.LogAttrKey(gwKey))
		return err
	}
	if err := b.deleteFrontend(gwKey, feName); err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to delete frontend",
			logging.LogAttrFrontendName(feName),
			logging.LogAttrError(err))
		return err
	}
	if b.firstSync {
		delete(b.frontendsContainedInFirstSync, feName)
	}
	return nil
}

func (b *HaproxyConfMgrImpl) deleteFrontend(gwKey client.ObjectKey, feName string) error {
	if err := b.configuration.deleteFrontend(b.logger, feName); err != nil {
		return err
	}
	b.frontendsOwnedbyGateway.RemoveFromUpdated(gwKey, feName)
	return nil
}

func (b *HaproxyConfMgrImpl) logGatewayUpdate(action string, gwKey k8stypes.NamespacedName) {
	b.logger.LogAttrs(context.Background(), slog.LevelDebug, "Processing Gateway ["+action+"]",
		logging.LogAttrKey(gwKey),
	)
}

func DeepCopyFrontend(original *models.Frontend) (*models.Frontend, error) {
	if original == nil {
		return nil, nil
	}
	var copied models.Frontend
	data, err := json.Marshal(original) // Serialize to JSON
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct
	return &copied, nil
}

func NewFrontendsOwnedbyGateway() FrontendsOwnedbyGateway {
	return FrontendsOwnedbyGateway{
		current: make(map[client.ObjectKey]map[string]struct{}),
		updated: make(map[client.ObjectKey]map[string]struct{}),
	}
}

func (f *FrontendsOwnedbyGateway) AddToUpdated(gwKey client.ObjectKey, frontendName string) {
	if _, ok := f.updated[gwKey]; !ok {
		f.updated[gwKey] = make(map[string]struct{})
	}
	f.updated[gwKey][frontendName] = struct{}{}
}

func (f *FrontendsOwnedbyGateway) RemoveFromUpdated(gwKey client.ObjectKey, frontendName string) {
	if _, ok := f.updated[gwKey]; ok {
		delete(f.updated[gwKey], frontendName)
	}
}

func (b *HaproxyConfMgrImpl) bindParams(_, bindName string, treeGw *tree.Gateway, treeListener *tree.Listener) models.BindParams {
	params := models.BindParams{}
	params.Name = bindName

	// If no TLS
	if treeListener.K8sResource.TLS == nil {
		return params
	}

	// TLS terminate
	listenerKey := tree.ListenerKey(treeGw.K8sResource, treeListener.K8sResource)
	certFileDir := b.params.certificateStorage.CertListPath(listenerKey)
	params.CrtList = certFileDir.FullPath()
	params.Ssl = true

	return params
}
