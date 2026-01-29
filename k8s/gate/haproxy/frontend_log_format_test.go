package haproxy

import (
	"io"
	"log/slog"
	"testing"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/store"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/tree"
	"k8s.io/apimachinery/pkg/types"
)

func TestReconcileFrontendLogFormatDoesNotPanicWithCapturedHeaders(t *testing.T) {
	controllerConfNsName := types.NamespacedName{
		Namespace: "haproxy-unified-gateway",
		Name:      "hugconf",
	}
	manager := HaproxyConfMgrImpl{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		params: HaproxyConfMgrParams{
			ControllerConfNsName: controllerConfNsName,
		},
		controllerStore: &tree.ControllerStore{
			ClusterStore: &store.ClusterStore{
				HugConfs: map[types.NamespacedName]*v3.HugConf{
					controllerConfNsName: {
						Spec: v3.ControllerConfSpec{
							HaproxyDefaults: &v3.HaproxyDefaults{
								LogFormat: "'%ci'",
								CaptureRequestHeaders: []v3.CaptureRequestHeader{
									{Name: "User-Agent", Length: 256},
								},
							},
						},
					},
				},
			},
		},
		configuration: Configuration{
			frontendLogFormat: "'%ci'",
			frontendCaptureHeaders: []captureRequestHeader{
				{Name: "User-Agent", Length: 256},
			},
		},
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("reconcileFrontendLogFormat panicked: %v", recovered)
		}
	}()

	changed := manager.reconcileFrontendLogFormat()

	if changed {
		t.Fatal("expected no change for identical captured headers")
	}
}
