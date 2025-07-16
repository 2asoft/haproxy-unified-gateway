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
package status

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/conditions"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	"k8s.io/apimachinery/pkg/util/wait"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *StatusUpdaterImpl) writeGatewayClassStatus(ctx context.Context, gwc *tree.GatewayClass) {
	updateOptions := StatusUpdateParams[*v1.GatewayClass]{
		Object:            gwc.K8sResource,
		DesiredConditions: gwc.Conditions,
		Getter:            s.cfg.client,
		StatusUpdater:     s.cfg.client.Status(),
		Logger:            s.cfg.logger,
		ConditionHandler:  &conditions.GatewayClassConditionImpl{},
		extractGVK:        s.cfg.extractGVK,
	}

	err := wait.ExponentialBackoffWithContext(
		ctx,
		wait.Backoff{
			Duration: time.Millisecond * 200,
			Factor:   2,
			Jitter:   0.5,
			Steps:    4,
			Cap:      time.Millisecond * 3000,
		},
		// Function returns true if the condition is satisfied, or an error if the loop should be aborted.
		TryUpdateStatusFunc(updateOptions),
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		s.cfg.logger.LogAttrs(context.Background(), slog.LevelError,
			"Failed to update status",
			logging.LogAttrResource(gwc.K8sResource, s.cfg.extractGVK(gwc.K8sResource)),
			logging.LogAttrError(err),
		)
	}
}
