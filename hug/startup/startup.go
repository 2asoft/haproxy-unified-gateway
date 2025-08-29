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
package startup

import (
	"context"

	"github.com/haproxytech/client-native/v6/configuration"
	cfgoptions "github.com/haproxytech/client-native/v6/configuration/options"
	"github.com/haproxytech/client-native/v6/models"
	md "github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/metadata"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/structured"
)

type ownerMetaData interface {
	*models.Frontend | *models.Backend
}

// StructuredFromFile read a initial configuration file and returns a Structured HAProxy configuration
// containing:
// Frontends/Backends
// that have the unified gateway metadata
// (the objects that the gateway manages)
func StructuredFromFile(cfgFile, transactionDir string) (structured.Structured, error) {
	confClient, err := configuration.New(context.Background(),
		cfgoptions.ConfigurationFile(cfgFile),
		cfgoptions.TransactionsDir(transactionDir),
		cfgoptions.UseMd5Hash,
	)
	if err != nil {
		return structured.Structured{}, err
	}

	_, backends, err := confClient.GetStructuredBackends("")
	if err != nil {
		return structured.Structured{}, err
	}
	_, frontends, err := confClient.GetStructuredFrontends("")
	if err != nil {
		return structured.Structured{}, err
	}

	structuredCfg := structured.Structured{
		Backends:  make(map[string]*models.Backend),
		Frontends: make(map[string]*models.Frontend),
	}
	for _, backend := range backends {
		if isUnifiedGatewayManaged(backend) {
			structuredCfg.Backends[backend.Name] = backend
		}
	}
	for _, frontend := range frontends {
		if isUnifiedGatewayManaged(frontend) {
			structuredCfg.Frontends[frontend.Name] = frontend
		}
	}

	return structuredCfg, nil
}

// isUnifiedGatewayManaged returns true if the object is managed by the Unified Gateway
// false otherwise
// based on the MetaData
func isUnifiedGatewayManaged[T ownerMetaData](obj T) bool {
	var metadata map[string]any
	switch o := any(obj).(type) {
	case *models.Frontend:
		metadata = o.Metadata
	case *models.Backend:
		metadata = o.Metadata
	default:
		return false
	}
	if _, ok := metadata[md.UnifiedGatewayMetatDataKey]; ok {
		return true
	}
	return false
}
