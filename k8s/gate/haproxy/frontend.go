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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/store"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type K8sObjectInfo struct {
	LinkID     string
	Generation int64
}

type FrontendMetaData map[string]map[string]K8sObjectInfo

func (b *HaproxyConfBuilderImpl) getFrontendName(gwKey k8stypes.NamespacedName, listener gatewayv1.Listener) (string, error) {
	tmpl, err := template.New("frontend").Parse(b.params.frontendNameTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse frontend name template: %w", err)
	}

	data := TemplateData{
		GATEWAY_NAMESPACE: gwKey.Namespace,
		GATEWAY_NAME:      gwKey.Name,
		LISTENER_NAME:     string(listener.Name),
		LINK_ID:           b.params.linkID,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (b *HaproxyConfBuilderImpl) buildFrontends() error {
	// Managed Gateways => Create / update/ delete frontends
	for gwKey, gateway := range b.controllerStore.GateTree.Gateways {
		switch gateway.TreeStatus.Status {
		case store.StatueUnchanged:
			continue
		case store.StatusUpserted:
			switch gateway.Valid {
			case true:
				b.logGatewayUpdate("upserted", gwKey)
				err := b.createOrUpdateFrontends(gwKey, gateway)
				if err != nil {
					return err
				}
			case false:
				b.logGatewayUpdate("invalid", gwKey)
				err := b.deleteFrontendForAllListeners(gwKey, gateway)
				if err != nil {
					return err
				}
			}

		case store.StatusDeleted:
			b.logGatewayUpdate("deleted", gwKey)
			err := b.deleteFrontendForAllListeners(gwKey, gateway)
			if err != nil {
				return err
			}
		}

		// Build HAProxy configuration for the gateway
	}

	// Unmanaged Gateways => Delete frontends
	for gwKey, gateway := range b.controllerStore.UnmanagedGateTree.Gateways {
		err := b.deleteFrontendForAllListeners(gwKey, gateway)
		if err != nil {
			return fmt.Errorf("failed to delete frontend for unmanaged gateway %s: %w", gwKey, err)
		}
	}
	return nil
}

func (b *HaproxyConfBuilderImpl) createOrUpdateFrontends(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	frontends := make(map[string]struct{})
	for _, listener := range gw.K8sResource.Spec.Listeners {
		newFe, err := b.newFrontend(gwKey, gw, listener)
		if err != nil {
			return err
		}
		b.createOrUpdateFrontend(newFe, frontends)
	}
	previousFrontends := b.frontendsByGateway[gwKey]
	if _, ok := b.frontendsByGateway[gwKey]; !ok {
		b.frontendsByGateway[gwKey] = make(map[string]struct{})
	}
	b.frontendsByGateway[gwKey] = frontends
	b.cleanupFrontends(gwKey, previousFrontends, frontends)

	return nil
}

func (b *HaproxyConfBuilderImpl) cleanupFrontends(gwKey client.ObjectKey, oldFrontends, newFrontends map[string]struct{}) {
	for frontendName := range oldFrontends {
		if _, ok := newFrontends[frontendName]; !ok {
			b.deleteFrontend(gwKey, frontendName)
		}
	}
}

func (b *HaproxyConfBuilderImpl) newFrontend(gwKey k8stypes.NamespacedName, treeGw *tree.Gateway, listener gatewayv1.Listener) (*models.Frontend, error) {
	// Create a frontend for each listener
	feName, err := b.getFrontendName(gwKey, listener)
	if err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError,
			"Failed to get frontend name",
			slog.String("frontendNameTemplate", b.params.frontendNameTemplate),
			logging.LogAttrKey(gwKey))
		return nil, fmt.Errorf("failed to get frontend name: %w", err)
	}

	md := b.frontendMetaData(treeGw)
	fe := &models.Frontend{
		FrontendBase: models.FrontendBase{
			Name:     feName,
			Metadata: md,
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
	if !b.params.disableIPv4 {
		bind := models.Bind{
			Port: &port,
			Address: func() string {
				if b.params.iPV4BindAddr != "" {
					return b.params.iPV4BindAddr
				}
				return "0.0.0.0"
			}(),
			BindParams: models.BindParams{Name: "v4"},
		}
		if fe.Binds == nil {
			fe.Binds = make(map[string]models.Bind)
		}
		fe.Binds[bind.Name] = bind
	}
	if !b.params.disableIPv6 {
		bind := models.Bind{
			Port: &port,
			Address: func() string {
				if b.params.iPV6BindAddr != "" {
					return b.params.iPV6BindAddr
				}
				return "::"
			}(),
			BindParams: models.BindParams{Name: "v6"},
		}
		if fe.Binds == nil {
			fe.Binds = make(map[string]models.Bind)
		}
		fe.Binds[bind.Name] = bind
	}
	return fe, nil
}

func (b *HaproxyConfBuilderImpl) createOrUpdateFrontend(newFe *models.Frontend, frontends map[string]struct{}) {
	if newFe == nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "nil frontend")
		return
	}
	defer func() {
		if newFe != nil {
			frontends[newFe.Name] = struct{}{}
		}
	}()

	if oldFe, ok := b.structuredCfgStore.Frontends[newFe.Name]; ok {
		// Check if they are the same
		if oldFe.Equal(*newFe) {
			b.logger.LogAttrs(context.Background(), slog.LevelDebug,
				"Frontend [same]",
				logging.LogAttrFrontendName(newFe.Name),
			)
			return
		}

		// Update existing frontend
		b.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Frontend [UPDATE]",
			logging.LogAttrFrontendName(newFe.Name),
		)
		b.structuredCfgStore.Frontends[newFe.Name] = newFe
		if _, ok := frontends[newFe.Name]; !ok {
			frontends[newFe.Name] = struct{}{}
		}
		// We need to deep copy the frontend to avoid modifying the original
		// as the diffs will be sent on a channel and used at the same time we continue to update the haproxy cfg store.
		deepCopied := DeepCopyFrontend(newFe)
		b.cfgDiffs.Updated.Frontends[newFe.Name] = deepCopied
	} else {
		// Create new frontend
		b.logger.LogAttrs(context.Background(), slog.LevelDebug,
			"Frontend [CREATE]",
			logging.LogAttrFrontendName(newFe.Name),
		)
		b.structuredCfgStore.Frontends[newFe.Name] = newFe
		if _, ok := frontends[newFe.Name]; !ok {
			frontends[newFe.Name] = struct{}{}
		}
		// We need to deep copy the frontend to avoid modifying the original
		// as the diffs will be sent on a channel and used at the same time we continue to update the haproxy cfg store.
		deepCopied := DeepCopyFrontend(newFe)
		b.cfgDiffs.Created.Frontends[newFe.Name] = deepCopied
	}
}

func (b *HaproxyConfBuilderImpl) deleteFrontendForAllListeners(gwKey k8stypes.NamespacedName, gw *tree.Gateway) error {
	// K8s resource might be in:
	k8sGateway := gw.GetK8sResource()
	if k8sGateway == nil {
		return fmt.Errorf("no K8s resource found for gateway %s", gwKey)
	}

	for _, listener := range k8sGateway.Spec.Listeners {
		// Frontend for each listener
		feName, err := b.getFrontendName(gwKey, listener)
		if err != nil {
			b.logger.LogAttrs(context.Background(), slog.LevelError,
				"Failed to get frontend name",
				slog.String("frontendNameTemplate", b.params.frontendNameTemplate),
				logging.LogAttrKey(gwKey))
			continue
		}
		b.deleteFrontend(gwKey, feName)
	}

	return nil
}

func (b *HaproxyConfBuilderImpl) deleteFrontend(gwKey client.ObjectKey, feName string) {
	// Retrieve the frontend from the store
	fe, ok := b.structuredCfgStore.Frontends[feName]
	if !ok {
		// It could happen that the frontend was already deleted
		// like gateway is:
		// - first unmanaged: Frontend is not created, not in store
		// - then deleted: Frontend is deleted, not in store
		return
	}
	// delete the frontend from the store
	b.logger.LogAttrs(context.Background(), slog.LevelDebug,
		"Frontend [DELETE]",
		logging.LogAttrFrontendName(feName),
	)
	delete(b.structuredCfgStore.Frontends, feName)
	delete(b.frontendsByGateway, gwKey)
	// We need to deep copy the frontend to avoid modifying the original
	// as the diffs will be sent on a channel and used at the same time we continue to update the haproxy cfg store.
	deepCopied := DeepCopyFrontend(fe)
	b.cfgDiffs.Deleted.Frontends[fe.Name] = deepCopied
}

func (b *HaproxyConfBuilderImpl) logGatewayUpdate(action string, gwKey k8stypes.NamespacedName) {
	b.logger.LogAttrs(context.Background(), slog.LevelDebug,
		"Haproxy Cfg processing Gateway ["+action+"]",
		logging.LogAttrKey(gwKey),
	)
}

func DeepCopyFrontend(original *models.Frontend) *models.Frontend {
	if original == nil {
		return nil
	}
	var copied models.Frontend
	data, _ := json.Marshal(original) // Serialize to JSON
	_ = json.Unmarshal(data, &copied) // Deserialize to a new struct
	return &copied
}

func (b *HaproxyConfBuilderImpl) frontendMetaData(treeGw *tree.Gateway) MetaData {
	fmd := make(FrontendMetaData)
	md := make(MetaData)
	md[MetatDataKey] = fmd
	k8sResource := treeGw.GetK8sResource()

	gvk := b.params.exctractGVK(k8sResource)
	fmd[gvk.Kind] = make(map[string]K8sObjectInfo)
	objKey := client.ObjectKeyFromObject(k8sResource)
	fmd[gvk.Kind][objKey.String()] = K8sObjectInfo{
		Generation: k8sResource.GetGeneration(),
		LinkID:     b.params.linkID,
	}

	return md
}
