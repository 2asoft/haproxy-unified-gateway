//
// Copyright 2025 HAProxy Technologies LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	gatecontroller "github.com/haproxytech/kubernetes-controller/k8s/gate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	opt "github.com/haproxytech/kubernetes-controller/k8s/gate/options"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kubectl/pkg/scheme"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func init() {
	utilruntime.Must(v3.AddToScheme(scheme.Scheme))
	utilruntime.Must(corev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(discoveryV1.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiext.AddToScheme(scheme.Scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(gwv1.Install(scheme.Scheme))
}

const (
	controllerNs = "haproxy-controller"
)

type Test struct {
	ctx       context.Context
	testEnv   *envtest.Environment
	g         *gomega.GomegaWithT
	cancel    context.CancelFunc
	namespace string
}

func NewTest(t *testing.T) (test Test, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	g := gomega.NewWithT(t)

	// Namespace
	namespace, err := setupNamespace()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../../api/definition",
			"../api",
		},
		ErrorIfCRDPathMissing: true,
	}

	test = Test{
		g:         g, // Gomega handle bound to `t`
		ctx:       ctx,
		cancel:    cancel,
		testEnv:   testEnv,
		namespace: namespace,
	}

	return test, nil
}

func (test *Test) StartTestEnv(t *testing.T) {
	// Bootstrapping test environment.
	cfg, err := test.testEnv.Start()
	test.g.Expect(err).ToNot(gomega.HaveOccurred())
	test.g.Expect(cfg).ToNot(gomega.BeNil())

	mgr, err := ctrlruntime.NewManager(cfg, ctrlruntime.Options{
		Scheme: scheme.Scheme,
	})
	test.g.Expect(err).ToNot(gomega.HaveOccurred())

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	test.g.Expect(err).ToNot(gomega.HaveOccurred())

	// Create controller namespace.
	gateNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: controllerNs,
		},
	}
	if err := client.Create(test.ctx, gateNs); err != nil {
		t.Fatalf("failed to create namespace: %s", err)
	}

	controllerConfig := config.ControllerPodConfig{}

	// Values to get from flags
	// to implement:  flags
	metricsConfig := config.MetricsConfig{
		Port:    6062,
		Enabled: false,
		Secure:  false,
	}
	// if gatewayClass =is empty, we will support all GatewayClasses that reference this controller
	// (through the spec.controllerName)
	gatewayClass := "haproxy"
	gatewayControllerName := "gate.haproxy.org/gateway-controller"
	controllerConfName := types.NamespacedName{
		Namespace: "test",
		Name:      "haproxyctrlconf",
	}

	whiteListNs := []string{"default", "kube-system", "haproxy-controller", "test", "test2"}
	// whiteListNs := []string{}

	// kubeconfig := testKubeConfig
	kubeconfig := ""

	syncPeriod := 1 * time.Second
	logLevel := slog.LevelDebug
	logCategories := []string{"all"}

	leaderElectionLockName := "kubernetes-controller-leader-election-lock"
	leaderElectionConfig := config.LeaderElectionConfig{
		Enabled:  false,
		LockName: leaderElectionLockName,
		Identity: controllerConfig.Name,
	}

	opts := []func(c *config.Configuration) error{
		opt.ControllerPodConfig(controllerConfig),
		opt.KubeConfig(kubeconfig),
		opt.GatewayClass(gatewayClass),
		opt.ControllerConf(controllerConfName),
		opt.SyncPeriod(syncPeriod),
		opt.MetricsConfig(metricsConfig),
		opt.LeaderElectionConfig(leaderElectionConfig),
		opt.ControllerName(gatewayControllerName),
		opt.WhiteListNamespaces(whiteListNs),
		opt.Logging(logLevel, logCategories),
	}
	gatecontrollercfg := config.Configuration{}

	for _, o := range opts {
		_ = o(&gatecontrollercfg)
	}

	err = gatecontroller.Add(test.ctx, gatecontrollercfg, mgr)
	test.g.Expect(err).ToNot(gomega.HaveOccurred())

	go func() {
		if err := mgr.Start(test.ctx); err != nil {
			t.Errorf("failed to start manager: %s", err)
			return
		}
	}()
}

func (test *Test) StopTestEnv(t *testing.T) {
	// Clean up and stop controller.
	test.cancel()

	// Tearing down the test environment.
	if err := test.testEnv.Stop(); err != nil {
		t.Fatalf("failed to stop testEnv: %s", err)
	}
}

func setupNamespace() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir = filepath.Base(dir)
	dir = strings.Map(func(r rune) rune {
		if r < 'a' || r > 'z' && r != '-' {
			return '-'
		}
		return r
	}, strings.ToLower(dir))
	return "e2e-tests-" + dir, nil
}
