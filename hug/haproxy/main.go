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
	"fmt"
	"log/slog"
	"sync"

	"github.com/haproxytech/client-native/v6/models"
	"github.com/haproxytech/kubernetes-controller/hug/haproxy/api"
	"github.com/haproxytech/kubernetes-controller/hug/haproxy/params"
	"github.com/haproxytech/kubernetes-controller/hug/haproxy/process"
	"github.com/haproxytech/kubernetes-controller/hug/reload"
	gatehaproxy "github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
)

type AppManager interface {
	Stop()
	Run()
}

type AppManagerImpl struct {
	client       api.HAProxyClient
	process      process.Process
	ctx          context.Context
	wg           *sync.WaitGroup
	logger       *slog.Logger
	haproxyCfgCh chan gatehaproxy.HaproxyConfDiffs
	params       params.Params
}

var _ AppManager = &AppManagerImpl{}

func NewAppManager(ctx context.Context, wg *sync.WaitGroup, cfgCh chan gatehaproxy.HaproxyConfDiffs,
	param params.Params,
	logger *slog.Logger,
) (AppManager, error) {
	mylogger := logger.With(logging.LogAttrCategory(logging.LogCategoryApp))

	haproxyClient, err := api.New(mylogger, param.CfgDir, param.MainCfgFile, param.HaproxyBinary, param.RuntimeSocket)
	if err != nil {
		err = fmt.Errorf("failed to initialize haproxy API client: %w", err)
		return nil, err
	}

	p := process.New(param, haproxyClient, logger)
	p.SetAPI(haproxyClient)

	reload.GetInstance().SetLogger(logger)

	return &AppManagerImpl{
		client:       haproxyClient,
		process:      p,
		ctx:          ctx,
		wg:           wg,
		logger:       mylogger,
		params:       param,
		haproxyCfgCh: cfgCh,
	}, nil
}

func (h *AppManagerImpl) Stop() {
	err := h.process.Service("stop")
	if err != nil {
		panic(err)
	}
}

func (h *AppManagerImpl) Run() {
	// Goroutine to listen on haproxyCfgCh and perform the haproxy configuration update
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case <-h.ctx.Done():
				h.logger.LogAttrs(context.Background(), slog.LevelInfo,
					"shutting down AppManager Run() goroutine",
				)
				return
			case haproxyCfg := <-h.haproxyCfgCh:
				err := h.applyCfgUpdates(haproxyCfg)
				if err != nil {
					h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to updated Haproxy config",
						logging.LogAttrError(err),
					)
				}
			}
		}
	}()
}

func (h *AppManagerImpl) applyCfgUpdates(diffs gatehaproxy.HaproxyConfDiffs) error {
	var err error
	if diffs.IsEmpty() {
		// Should not happen, already checked before
		return nil
	}
	// Process the received HaproxyConfDiffs
	h.logger.LogAttrs(context.Background(), slog.LevelDebug,
		"Starting processing HaproxyConfDiffs",
		slog.String("HaproxyConfDiffs", fmt.Sprintf("%+v", diffs.Stats())))
	// Log, send result to the controller to update status
	defer func() {
		h.confUpdateProcessed(diffs, err)
	}()

	// -----------------
	// Start transaction

	err = h.client.APIStartTransaction()
	if err != nil {
		h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to start transaction",
			logging.LogAttrError(err),
		)
		return err
	}

	if err = h.processCreate(diffs.Created); err != nil {
		return err
	}
	if err = h.processUpdate(diffs.Updated); err != nil {
		return err
	}
	if err = h.processDelete(diffs.Deleted); err != nil {
		return err
	}

	// -----------------
	// Commit transaction

	err = h.client.APIFinalCommitTransaction()
	if err != nil {
		h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to commit transaction",
			logging.LogAttrError(err),
		)
		return err
	}

	// -----------------
	// Reload ?
	if reload.GetInstance().NeedReload() {
		h.logger.LogAttrs(context.Background(), slog.LevelInfo,
			"Haproxy reload")
		if err = h.process.Service("reload"); err != nil {
			h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to reload Haproxy",
				logging.LogAttrError(err),
			)
			return err
		}
		h.logger.LogAttrs(context.Background(), slog.LevelInfo,
			"Haproxy reloaded")
	}

	return nil
}

func (h *AppManagerImpl) processCreate(created gatehaproxy.Structured) error {
	for _, createdFE := range created.Frontends {
		if createdFE == nil {
			// Should not happend
			h.logger.LogAttrs(context.Background(), slog.LevelError, "nil frontend")
			continue
		}

		err := h.client.FrontendCreate(*createdFE)
		if err != nil {
			h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to create frontend",
				logging.LogAttrError(err),
			)
			return err
		}
	}
	// To do for BE
	// ....
	return nil
}

func (h *AppManagerImpl) processDelete(deleted gatehaproxy.Structured) error {
	for _, deletedFE := range deleted.Frontends {
		if deletedFE == nil {
			// Should not happend
			h.logger.LogAttrs(context.Background(), slog.LevelError, "nil frontend")
			continue
		}

		err := h.client.FrontendDelete(deletedFE.Name)
		if err != nil {
			h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to create frontend",
				logging.LogAttrError(err),
			)
			return err
		}
	}
	// To do for BE
	// ....
	return nil
}

func (h *AppManagerImpl) processUpdate(updated gatehaproxy.Structured) error {
	for _, udpatedFE := range updated.Frontends {
		if udpatedFE == nil {
			// Should not happend
			h.logger.LogAttrs(context.Background(), slog.LevelError, "nil frontend")
			continue
		}

		err := h.client.FrontendEdit(*udpatedFE)
		if err != nil {
			h.logger.LogAttrs(context.Background(), slog.LevelError, "failed to create frontend",
				logging.LogAttrError(err),
			)
			return err
		}
	}
	// To do for BE
	// ....
	return nil
}

func (h *AppManagerImpl) confUpdateProcessed(diffs gatehaproxy.HaproxyConfDiffs, err error) {
	h.client.APIDisposeTransaction()

	result := gatehaproxy.HaproxyConfUpdateResult{
		Error:                   err,
		UpdatedSectionsMetaData: make(map[string]any),
	}

	// Frontends
	for _, fe := range diffs.Created.Frontends {
		addFrontendMetadataToResult(result.UpdatedSectionsMetaData, fe)
	}
	for _, fe := range diffs.Updated.Frontends {
		addFrontendMetadataToResult(result.UpdatedSectionsMetaData, fe)
	}
	for _, fe := range diffs.Deleted.Frontends {
		addFrontendMetadataToResult(result.UpdatedSectionsMetaData, fe)
	}
	// Backends
	for _, be := range diffs.Created.Backends {
		addBackendMetadataToResult(result.UpdatedSectionsMetaData, be)
	}
	for _, be := range diffs.Updated.Backends {
		addBackendMetadataToResult(result.UpdatedSectionsMetaData, be)
	}
	for _, be := range diffs.Deleted.Backends {
		addBackendMetadataToResult(result.UpdatedSectionsMetaData, be)
	}

	h.logger.LogAttrs(context.Background(), slog.LevelInfo, "Haproxy configuration update result",
		slog.Any("result", result))
}

func addFrontendMetadataToResult(meta map[string]any, fe *models.Frontend) {
	if fe != nil {
		meta[fe.Name] = fe.Metadata
	}
}

func addBackendMetadataToResult(meta map[string]any, be *models.Backend) {
	if be != nil {
		meta[be.Name] = be.Metadata
	}
}
