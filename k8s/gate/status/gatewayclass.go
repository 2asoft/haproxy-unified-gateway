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
	"github.com/haproxytech/kubernetes-controller/k8s/gate/tree"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

func (s *StatusUpdaterImpl) writeGatewayClassStatus(ctx context.Context,
	gwc *tree.GatewayClass,
) {
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
		tryUpdateGatewayClassStatusFunc(s.cfg.client, s.cfg.client.Status(), gwc, *s.cfg.logger),
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		s.cfg.logger.Error(
			"Failed to update status",
			"namespace", gwc.K8sResource.GetNamespace(),
			"name", gwc.K8sResource.GetName(),
			"kind", s.cfg.extractGVK(gwc.K8sResource),
			"error", err)
	}
}

func tryUpdateGatewayClassStatusFunc(getter client.Reader, statusUpdater client.SubResourceWriter, gwc *tree.GatewayClass, logger slog.Logger) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		// First get the latest version of the object to avoid conflict
		currentObj := &v1.GatewayClass{}
		err := getter.Get(ctx, types.NamespacedName{Namespace: gwc.K8sResource.GetNamespace(), Name: gwc.K8sResource.GetName()}, currentObj)
		if err != nil {
			// apierrors.IsNotFound(err) can happen when the resource is deleted,
			// so no need to retry or return an error.
			if apierrors.IsNotFound(err) {
				return true, nil
			}

			logger.Info(
				"Encountered error when getting resource to update status",
				"error", err,
				"object", client.ObjectKeyFromObject(gwc.K8sResource),
				"kind", gwc.K8sResource.GetObjectKind().GroupVersionKind().Kind,
			)

			return false, nil
		}

		// Check if the object status is already up to date
		currentConditions := conditions.NewConditionsFromMetav1Conditions(currentObj.Status.Conditions)
		if currentConditions.Equal(gwc.Conditions) {
			logger.Info(
				"Status already up to date",
				"object", client.ObjectKeyFromObject(gwc.K8sResource),
				"kind", currentObj.GetObjectKind().GroupVersionKind().Kind,
				"status", currentObj.Status.Conditions[0].Reason,
			)
			return true, nil
		}
		newConditions := gwc.Conditions.ToMetav1Conditions()
		newStatus := v1.GatewayClassStatus{
			Conditions: newConditions,
		}

		currentObj.Status = newStatus
		if err := statusUpdater.Update(ctx, currentObj); err != nil {
			logger.Info(
				"Encountered error updating status",
				"error", err,
				"object", client.ObjectKeyFromObject(gwc.K8sResource),
				"kind", currentObj.GetObjectKind().GroupVersionKind().Kind,
			)

			return false, nil
		} else {
			logger.Info(
				"Successfully updated status",
				"object", client.ObjectKeyFromObject(gwc.K8sResource),
				"kind", currentObj.GetObjectKind().GroupVersionKind().Kind,
			)
		}

		return true, nil
	}
}
