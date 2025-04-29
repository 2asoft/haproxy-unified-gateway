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
package events

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/index"

	discoveryV1 "k8s.io/api/discovery/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventHandler handles events.
type EventHandler interface {
	// HandleEventBatch handles a batch of events.
	// EventBatch can include duplicated events.
	HandleEventBatch(ctx context.Context, logger *slog.Logger, batch EventBatch)
}

// EventHandlerConfig holds configuration parameters for eventHandlerImpl.
type EventHandlerConfig struct {
	// k8sClient is a Kubernetes API client.
	k8sClient client.Client
	// k8sReader is a Kubernets API reader.
	k8sReader client.Reader
	// controllerPodConfig contains information about this Pod.
	controllerPodConfig config.ControllerPodConfig
	// gatewayCtlrName is the name of the controller.
	gatewayCtlrName string
}

// eventHandlerImpl implements EventHandler.
// eventHandlerImpl is responsible for:
// - Reconciling the Gateway API and Kubernetes built-in resources with the HAProxy configuration.
// - Keeping the statuses of the Gateway API resources updated.
type eventHandlerImpl struct {
	cfg EventHandlerConfig
}

// NewEventHandlerImpl creates a new eventHandlerImpl.
func NewEventHandlerImpl(cfg EventHandlerConfig) *eventHandlerImpl {
	handler := &eventHandlerImpl{
		cfg: cfg,
	}

	return handler
}

// NewEventHandlerImpl creates a new eventHandlerImpl.
func NewEventHandlerConfig(
	k8sClient client.Client,
	k8sReader client.Reader,
	controllerPodConfig config.ControllerPodConfig,
	gatewayCtlrName string,
) EventHandlerConfig {
	eventHandlerConfig := EventHandlerConfig{
		k8sClient:           k8sClient,
		k8sReader:           k8sReader,
		controllerPodConfig: controllerPodConfig,
		gatewayCtlrName:     gatewayCtlrName,
	}
	return eventHandlerConfig
}

func (h *eventHandlerImpl) HandleEventBatch(ctx context.Context, logger *slog.Logger, batch EventBatch) {
	start := time.Now()
	logger.Info("Started processing event batch", "len", len(batch.Events))

	defer func() {
		duration := time.Since(start)
		logger.Info(
			"Finished processing event batch",
			"duration", duration.String(),
		)
	}()

	for _, e := range batch.Events {
		switch obj := e.(type) {
		case *UpsertEvent:
			logger.Info("Processing event in batch", "eventType", "upsert", "GVK",
				obj.Resource.GetObjectKind().GroupVersionKind(), "resource", client.ObjectKeyFromObject(obj.Resource))
		case *DeleteEvent:
			logger.Info("Processing event in batch", "eventType", "delete", "GVK",
				obj.Type.GetObjectKind().GroupVersionKind(), "resource", obj.NamespacedName)
		}
	}

	// Update some sort of store
	// Compute config
	// Compute diffs....
	// What we need to do has to be done

	// START EXAMPLE
	// Below is just an example
	// We list EndpointSlices using the Service Name Index Field we added as an index to the EndpointSlice cache.
	// This allows us to perform a quick lookup of all EndpointSlices for a Service.
	var endpointSliceList discoveryV1.EndpointSliceList
	svcName := "http-echo"
	svcNs := "default"
	err := h.cfg.k8sClient.List(
		ctx,
		&endpointSliceList,
		client.MatchingFields{index.EndpointSliceServiceNameIndexField: svcName},
		client.InNamespace(svcNs),
	)
	if err != nil {
		logger.Error("error client.List svc http-echo", "error", err)
	}
	logger.Info(
		fmt.Sprintf("JUST AN EXAMPLE to show cache indexes usage. endpoints for http-echo svc %v", endpointSliceList))
	// END EXAMPLE

	h.updateHAProxy(ctx, logger)  //revive:disable:unused-parameters
	h.updateStatuses(ctx, logger) //revive:disable:unused-parameter
}

func (*eventHandlerImpl) updateStatuses(ctx context.Context, logger *slog.Logger) {
}

// Will need some sort of store!
func (*eventHandlerImpl) updateHAProxy(ctx context.Context, logger *slog.Logger) {
}
