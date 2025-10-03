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
package api

import (
	"context"
	"log/slog"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/kubernetes-controller/hug/reload"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
)

func (c *clientNative) BackendCreate(backend models.Backend) error {
	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return err
	}
	b := &models.Backend{BackendBase: backend.BackendBase}
	errCreate := configuration.CreateBackend(b, c.activeTransaction, 0)
	if errCreate != nil {
		// ... maybe it's already existing, so just edit it.
		if err := configuration.EditBackend(backend.Name, &backend, c.activeTransaction, 0); err != nil {
			c.logger.LogAttrs(context.Background(), slog.LevelError, "failed to edit backend",
				logging.LogAttrError(err),
				slog.String("backend", backend.Name),
			)
			return err
		}
	}
	reload.Instance().SetReload("Backend upserted %s", backend.Name)

	return err
}

func (c *clientNative) BackendDelete(backendName string) error {
	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return err
	}
	reload.Instance().SetReload("Backend deleted %s", backendName)
	return configuration.DeleteBackend(backendName, c.activeTransaction, 0)
}

func (c *clientNative) BackendsGet() (models.Backends, error) {
	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return nil, err
	}
	// TODO: complete with children
	_, backends, err := configuration.GetBackends(c.activeTransaction)

	return backends, err
}

func (c *clientNative) BackendGet(backendName string) (models.Backend, error) {
	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return models.Backend{}, err
	}
	// TODO: complete with children
	_, backend, err := configuration.GetBackend(backendName, c.activeTransaction)
	if err != nil {
		return models.Backend{}, err
	}

	return *backend, err
}

func (c *clientNative) BackendEdit(backend models.Backend) error {
	configuration, err := c.nativeAPI.Configuration()
	if err != nil {
		return err
	}
	b := &models.Backend{BackendBase: backend.BackendBase}
	if err := configuration.EditBackend(backend.Name, b, c.activeTransaction, 0); err != nil {
		c.logger.LogAttrs(context.Background(), slog.LevelError, "failed to edit backend",
			logging.LogAttrError(err),
			slog.String("frontend", backend.Name),
		)
		return err
	}

	reload.Instance().SetReload("Backend upserted %s", backend.Name)
	return nil
}
