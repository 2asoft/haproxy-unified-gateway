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
	"context"
	"encoding/json"
	"log/slog"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/haproxy/storage"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/protocols"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/tree"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/utils"
)

func (b *HaproxyConfMgrImpl) getFrontendName(vListenerName string) string {
	return b.params.LinkID + "_" + vListenerName
}

func (b *HaproxyConfMgrImpl) processVirtualListener() error {
	var errors utils.Errors
	// VirtualListeners are mapped 1 to 1 with Frontends,
	// so we can directly create/update/delete Frontends while processing VirtualListeners
	for vlName, vListener := range b.controllerStore.GateTree.VirtualListeners {
		switch vListener.Status {
		case store.StatusUnchanged:
			continue
		case store.StatusUpserted:
			err := b.onUpsertedVirtualListener(vlName, vListener)
			errors.Add(err)
		case store.StatusDeleted:
			err := b.onDeletedVirtualListener(vlName)
			errors.Add(err)
		}
	}

	return errors.Result()
}

func (b *HaproxyConfMgrImpl) onUpsertedVirtualListener(vlName string, vListener *tree.VirtualListener) error {
	b.logVirtualListenerUpdate("upserted", vlName)
	err := b.upsertFrontends(vlName, vListener)
	return err
}

func (b *HaproxyConfMgrImpl) onDeletedVirtualListener(vlName string) error {
	b.logVirtualListenerUpdate("deleted", vlName)
	err := b.deleteFrontendForVirtualListener(vlName)
	return err
}

func (b *HaproxyConfMgrImpl) upsertFrontends(vListenerName string, vListener *tree.VirtualListener) error {
	newFe, err := b.newFrontend(vListenerName, vListener)
	if err != nil {
		return err
	}
	if b.firstSync.flag {
		b.firstSync.frontends[newFe.Name] = struct{}{}
	}
	if err := b.configuration.upsertFrontend(b.logger, newFe); err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to upsert frontend",
			logging.LogAttrFrontendName(newFe.Name),
			logging.LogAttrError(err))
	}

	return nil
}

func (b *HaproxyConfMgrImpl) newFrontend(vListenerName string, vListener *tree.VirtualListener) (*models.Frontend, error) { //revive:disable:function-length
	// Create a frontend for each listener
	frontendName := b.getFrontendName(vListenerName)

	md := b.metadataManager.FrontendMetaData(vListener)

	pathExactMap := b.params.mapsStorage.MapPath(frontendName, storage.PATH_EXACT_MAP)
	pathPrefixMap := b.params.mapsStorage.MapPath(frontendName, storage.PATH_PREFIX_MAP)
	pathDomainWPathExactMap := b.params.mapsStorage.MapPath(frontendName, storage.PATH_EXACT_DOMAIN_WILDCARD_MAP)
	pathRegexMap := b.params.mapsStorage.MapPath(frontendName, storage.PATH_REGEX_MAP)
	sniMap := b.params.mapsStorage.MapPath(frontendName, storage.SNI_MAP)
	sniDomainWildcardMap := b.params.mapsStorage.MapPath(frontendName, storage.SNI_DOMAIN_WILDCARD_MAP)

	b.params.mapsStorage.EnsureMapData(pathExactMap)
	b.params.mapsStorage.EnsureMapData(pathPrefixMap)
	b.params.mapsStorage.EnsureMapData(pathRegexMap)
	b.params.mapsStorage.EnsureMapData(pathDomainWPathExactMap)
	b.params.mapsStorage.EnsureMapData(sniMap)
	b.params.mapsStorage.EnsureMapData(sniDomainWildcardMap)

	var tcpRules []*models.TCPRequestRule
	var httpRules []*models.HTTPRequestRule
	var backendSwitchingRules []*models.BackendSwitchingRule
	var aclList []*models.ACL
	switch {
	case vListener.ProtocolCategory == protocols.ProtocolCategoryTLS:
		// TLS Passthrough
		tcpRules = []*models.TCPRequestRule{
			{ // tcp-request content reject if !{ req.ssl_hello_type 1 }
				Type:     "content",
				Action:   "reject",
				Cond:     "if",
				CondTest: "!{ req.ssl_hello_type 1 }",
			},
			{
				// tcp-request inspect-delay 50000
				Type:    "inspect-delay",
				Timeout: new(int64(50000)),
			},
			{
				// tcp-request content set-var(sess.sni) req.ssl_sni
				Type:     "content",
				Action:   "set-var",
				VarName:  "sni",
				VarScope: "sess",
				Expr:     "req.ssl_sni",
			},
			{
				// tcp-request content set-var(txn.sni_match) req.ssl_sni,map(sni.map)
				Type:     "content",
				Action:   "set-var",
				VarName:  "sni_match",
				VarScope: "txn",
				Expr:     "req.ssl_sni,map(" + sniMap.FullPath() + ")",
			},
			{
				// tcp-request content set-var(txn.sni_match,ifnotexists) req.ssl_sni,map_end(sniDomainWildcardMap.map)
				Type:     "content",
				Action:   "set-var",
				VarName:  "sni_match,ifnotexists",
				VarScope: "txn",
				Expr:     "req.ssl_sni,map_end(" + sniDomainWildcardMap.FullPath() + ")",
			},
		}
		backendSwitchingRules = []*models.BackendSwitchingRule{
			{
				Name:     "%[var(txn.backend)]",
				Cond:     "if",
				CondTest: "route_is_json",
			},
			{
				Name: "%[var(txn.sni_match),field(1,.)]",
			},
		}
		aclList = []*models.ACL{
			{ // acl route_is_json var(txn.sni_match),bytes(0,1) -m str
				ACLName:   "route_is_json",
				Criterion: "var(txn.sni_match),bytes(0,1)",
				Value:     "-m str {",
				Metadata: map[string]any{
					"hug": "for lua routing",
				},
			},
		}

	default:
		httpRules = []*models.HTTPRequestRule{
			{ // http-request set-var(txn.path) path
				Type:     "set-var",
				VarName:  "path",
				VarScope: "txn",
				VarExpr:  "path",
			},
			{ // http-request set-var(txn.host) req.hdr(Host),host_only
				Type:     "set-var",
				VarName:  "host",
				VarScope: "txn",
				VarExpr:  "req.hdr(Host),host_only",
			},
			{ // http-request set-var(txn.base) var(txn.host),concat("",txn.path)
				Type:     "set-var",
				VarName:  "base",
				VarScope: "txn",
				VarExpr:  "var(txn.host),concat(\"\",txn.path)",
			},
			{
				// exact domain + exact path
				// http-request set-var(txn.route) base,map(route_exact_match.map)
				Type:     "set-var",
				VarName:  "route",
				VarScope: "txn",
				VarExpr:  "var(txn.base),map(" + pathExactMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "exact domain + exact path"},
			},
			{
				// # any domain + exact path
				// http-request set-var(txn.route,ifnotexists) path,map(route_exact_match.map)
				Type:     "set-var",
				VarName:  "route,ifnotexists",
				VarScope: "txn",
				VarExpr:  "path,map(" + pathExactMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "any domain + exact path"},
			},
			{
				// # exact domain + path prefix
				// http-request set-var(txn.route,ifnotexists) base,map_beg(route_prefix_match.map)
				Type:     "set-var",
				VarName:  "route,ifnotexists",
				VarScope: "txn",
				VarExpr:  "var(txn.base),map_beg(" + pathPrefixMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "exact domain + path prefix"},
			},
			{
				//  # any domain + path prefix
				// http-request set-var(txn.route,ifnotexists) path,map_beg(route_prefix_match.map)
				Type:     "set-var",
				VarName:  "route,ifnotexists",
				VarScope: "txn",
				VarExpr:  "path,map_beg(" + pathPrefixMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "any domain + path prefix"},
			},
			{
				//	# domain wildcard + exact path
				//	 http-request set-var(txn.route,ifnotexists) base,map_end(route_dw_ep.map)
				Type:     "set-var",
				VarName:  "route,ifnotexists",
				VarScope: "txn",
				VarExpr:  "var(txn.base),map_end(" + pathDomainWPathExactMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "domain wildcard + exact path"},
			},
			{
				// # any domain + path regex
				// http-request set-var(txn.route,ifnotexists) path,map_reg(route_regex.map) # ^/(foo|bar)/.*
				Type:     "set-var",
				VarName:  "route,ifnotexists",
				VarScope: "txn",
				VarExpr:  "path,map_reg(" + pathRegexMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "any domain + path regex"},
			},
			{
				// # domain wildcard + path prefix. Example: ^[^.]+\.domain\.com/v1/foo/.*   # or map_sub
				// # domain wildcard + path regex   Example: ^[^.]+\.domain\.com/v[1-3]/foo
				// # exact domain + path regex      Example: ^www\.domain\.com/v[1-3]/foo
				// http-request set-var(txn.route,ifnotexists) base,map_reg(route_regex.map)
				Type:     "set-var",
				VarName:  "route,ifnotexists",
				VarScope: "txn",
				VarExpr:  "var(txn.base),map_reg(" + pathRegexMap.FullPath() + ")",
				Metadata: map[string]any{"hug": "domain wildcard + path prefix or regex, exact domain + path regex"},
			},
			{
				// http-request lua.route if route_is_json
				Type:      "lua",
				LuaAction: "route",
				Cond:      "if",
				CondTest:  "route_is_json",
				Metadata: map[string]any{
					"hug": "lua routing",
				},
			},
		}
		backendSwitchingRules = []*models.BackendSwitchingRule{
			{
				Name:     "%[var(txn.backend)]",
				Cond:     "if",
				CondTest: "route_is_json",
			},
			{
				Name: "%[var(txn.route)]",
			},
		}
		aclList = []*models.ACL{
			{ // acl route_is_json var(txn.route),bytes(0,1) -m str {
				ACLName:   "route_is_json",
				Criterion: "var(txn.route),bytes(0,1)",
				Value:     "-m str {",
				Metadata: map[string]any{
					"hug": "for lua routing",
				},
			},
		}
	}

	fe := &models.Frontend{
		FrontendBase: models.FrontendBase{
			Name:           frontendName,
			From:           b.params.DefaultsSectionName,
			Metadata:       md,
			DefaultBackend: "backend_not_found",
			Mode: func() string {
				if vListener.ProtocolCategory == protocols.ProtocolCategorySecure || vListener.ProtocolCategory == protocols.ProtocolCategoryInsecure {
					return "http"
				}
				if vListener.ProtocolCategory == protocols.ProtocolCategoryTLS {
					return "tcp"
				}
				return ""
			}(),
		},
		ACLList:                  aclList,
		TCPRequestRuleList:       tcpRules,
		HTTPRequestRuleList:      httpRules,
		BackendSwitchingRuleList: backendSwitchingRules,
	}

	// Set other frontend properties based on the listener
	port := int64(vListener.Port)
	if !b.params.DisableIPv4 {
		bind := models.Bind{
			Port: &port,
			Address: func() string {
				if b.params.IPv4BindAddress != "" {
					return b.params.IPv4BindAddress
				}
				return "0.0.0.0"
			}(),
			BindParams: b.bindParams(frontendName, "v4", vListenerName, vListener),
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
			BindParams: b.bindParams(frontendName, "v6", vListenerName, vListener),
		}
		if fe.Binds == nil {
			fe.Binds = make(map[string]models.Bind)
		}
		fe.Binds[bind.Name] = bind
	}
	return fe, nil
}

func (b *HaproxyConfMgrImpl) deleteFrontendForVirtualListener(virtualListenerName string) error {
	// Frontend for each listener
	feName := virtualListenerName
	if err := b.deleteFrontend(feName); err != nil {
		b.logger.LogAttrs(context.Background(), slog.LevelError, "Failed to delete frontend",
			logging.LogAttrFrontendName(feName),
			logging.LogAttrError(err))
		return err
	}
	if b.firstSync.flag {
		delete(b.firstSync.frontends, feName)
	}
	return nil
}

func (b *HaproxyConfMgrImpl) deleteFrontend(feName string) error {
	return b.configuration.deleteFrontend(b.logger, feName)
}

func (b *HaproxyConfMgrImpl) logVirtualListenerUpdate(action string, virtualListenerName string) {
	b.logger.LogAttrs(context.Background(), slog.LevelDebug, "Processing VirtualListener ["+action+"]",
		logging.LogAttrVirtualListenerName(virtualListenerName),
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

func (b *HaproxyConfMgrImpl) bindParams(_, bindName string, vListenerName string, vListener *tree.VirtualListener) models.BindParams {
	params := models.BindParams{}
	params.Name = bindName

	// If no TLS
	if vListener.ProtocolCategory == protocols.ProtocolCategoryInsecure || vListener.ProtocolCategory == protocols.ProtocolCategoryTLS {
		return params
	}

	// TLS terminate
	if vListener.ProtocolCategory == protocols.ProtocolCategorySecure {
		certFileDir := b.params.certificateStorage.CertListPath(vListenerName)
		params.CrtList = certFileDir.FullPath()
		params.Ssl = true
	}
	return params
}
