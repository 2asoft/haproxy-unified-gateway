package status

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
	objtypes "github.com/haproxytech/haproxy-unified-gateway/k8s/gate/object-types"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/tree"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func (s *StatusUpdaterImpl) writeTLSRouteStatus(ctx context.Context, route *tree.TLSRoute) {
	updateOptions := StatusUpdateParams[*v1alpha2.TLSRoute]{
		Object:        objtypes.ObjectTypeTLSRoute,
		NsName:        types.NamespacedName{Name: route.K8sResource.Name, Namespace: route.K8sResource.Namespace},
		StatusPatcher: newTLSRouteStatusPatcher(route),
		Getter:        s.config.client,
		StatusUpdater: s.config.client.Status(),
		Logger:        s.config.logger,
		extractGVK:    s.config.extractGVK,
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
		TryPatchStatusFunc(updateOptions),
	)
	if err != nil && !errors.Is(err, context.Canceled) {
		s.config.logger.LogAttrs(context.Background(), slog.LevelError,
			"Failed to update status",
			logging.LogAttrResource(route.K8sResource, s.config.extractGVK(route.K8sResource)),
			logging.LogAttrError(err),
		)
	}
}
