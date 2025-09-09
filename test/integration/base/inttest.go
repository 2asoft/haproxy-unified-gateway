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

package base

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"

	"github.com/haproxytech/client-native/v6/models"
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	gate "github.com/haproxytech/kubernetes-controller/k8s/gate"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/structured"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	opt "github.com/haproxytech/kubernetes-controller/k8s/gate/options"
	"github.com/haproxytech/kubernetes-controller/test/integration/utils"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryV1 "k8s.io/api/discovery/v1"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func init() {
	utilruntime.Must(v3.AddToScheme(scheme.Scheme))
	utilruntime.Must(corev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(discoveryV1.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiext.AddToScheme(scheme.Scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(gatewayv1.Install(scheme.Scheme))
}

const (
	controllerNs = "haproxy-controller"
)

var hugConfNsName = types.NamespacedName{
	Namespace: "test",
	Name:      "hugconf",
}

type IntTest struct {
	Ctx       context.Context
	Client    ctrlruntimeclient.Client
	TestEnv   *envtest.Environment
	cancel    context.CancelFunc
	Namespace string
}

func NewIntTest(t *testing.T) (test IntTest, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	g := gomega.NewWithT(t)

	// Namespace
	namespace, err := utils.GetIntTestNamespace()
	g.Expect(err).ToNot(gomega.HaveOccurred())

	testEnvVersion := os.Getenv("ENVTEST_VERSION")
	installPath := os.Getenv("KUBEBUILDER_ASSETS")

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../../api/definition",
			"../api",
		},
		ErrorIfCRDPathMissing:       true,
		DownloadBinaryAssets:        true,
		DownloadBinaryAssetsVersion: testEnvVersion,
		BinaryAssetsDirectory:       installPath,
	}

	test = IntTest{
		Ctx:       ctx,
		cancel:    cancel,
		TestEnv:   testEnv,
		Namespace: namespace,
	}

	return test, nil
}

func (test *IntTest) StartTestEnv(t *testing.T) {
	// Bootstrapping test environment.
	cfg, err := test.TestEnv.Start()
	g := gomega.NewWithT(t)

	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(cfg).ToNot(gomega.BeNil())
	// metricsConfig := config.MetricsConfig{
	// 	Port:    6062,
	// 	Enabled: false,
	// 	Secure:  false,
	// }
	kubeconfigPath, err := WriteKubeconfig(cfg)
	g.Expect(err).ToNot(gomega.HaveOccurred())
	t.Logf("kubeconfig path: %s", kubeconfigPath)

	mgr, err := ctrlruntime.NewManager(cfg, ctrlruntime.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	test.Client = client
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Create the test Namespace
	err = test.createNamespace(test.Namespace)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Create controller namespace.
	err = test.createNamespace(controllerNs)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Values to get from flags
	// to implement:  flags

	// if gatewayClass =is empty, we will support all GatewayClasses that reference this controller
	// (through the spec.controllerName)
	controllerName := "gate.haproxy.org/hug"

	logLevels := map[v3.Category]slog.Level{
		logging.LogCategoryK8s:           slog.LevelWarn,
		logging.LogCategoryGate:          slog.LevelDebug,
		logging.LogCategoryApp:           slog.LevelDebug,
		logging.LogCategoryHaproxyCfgMgr: slog.LevelDebug,
		logging.LogCategoryBatch:         slog.LevelInfo,
		logging.LogCategoryStatus:        slog.LevelDebug,
		logging.LogCategoryReloadMgr:     slog.LevelInfo,
		logging.LogCategoryCertsStorage:  slog.LevelDebug,
	}

	syncPeriod := 1 * time.Second

	opts := []func(c *config.Configuration) error{
		//	opt.KubeConfig(kubeconfig),
		opt.ControllerConfCRD(hugConfNsName),
		opt.SyncPeriod(syncPeriod),
		opt.ControllerName(controllerName),
		opt.Logging(logging.LogHandlerTypeText, logging.DefaultLevel, logLevels),
		opt.InitialStructured(structured.Structured{
			Backends:  make(map[string]*models.Backend),
			Frontends: make(map[string]*models.Frontend),
		}),
		opt.DefaultsSectionName(config.DefaultsSectionName),
		opt.Namespaces([]string{test.Namespace}),
		opt.LinkID("linkid"),
	}
	gatecontrollercfg := config.Configuration{}
	gatecontrollercfg.ApplyDefaults()

	for _, o := range opts {
		_ = o(&gatecontrollercfg)
	}
	logrLoggerFromSlog := logr.FromSlogHandler(gatecontrollercfg.LogHandler)
	ctrlruntime.SetLogger(logrLoggerFromSlog)

	err = gate.Add(test.Ctx, gatecontrollercfg, nil, mgr)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	go func() {
		if err := mgr.Start(test.Ctx); err != nil {
			t.Errorf("failed to start manager: %s", err)
			return
		}
	}()
}

func (test *IntTest) StopTestEnv(t *testing.T) {
	g := gomega.NewWithT(t)

	// delete test Namespace
	err := test.cleanupNamespace(test.Namespace)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// delete the controller namespace
	err = test.cleanupNamespace(controllerNs)
	g.Expect(err).ToNot(gomega.HaveOccurred())

	// Clean up and stop controller.
	test.cancel()

	// Tearing down the test environment.
	if err := test.TestEnv.Stop(); err != nil {
		t.Fatalf("failed to stop testEnv: %s", err)
	}
}

func (test *IntTest) createNamespace(ns string) error {
	err := utils.CreateRuntimeObject(test.Ctx, test.Client, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}, true)
	return err
}

func (test *IntTest) cleanupNamespace(ns string) error {
	err := utils.DeleteRuntimeObject(test.Ctx, test.Client, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}, true)
	return err
}

// WriteKubeconfig writes the given rest.Config to a kubeconfig file.
// It returns the path to the file and an error if it fails.
func WriteKubeconfig(cfg *rest.Config) (string, error) {
	// Create a clientcmdapi.Config object from the rest.Config
	clusters := make(map[string]*api.Cluster)
	clusters["envtest-cluster"] = &api.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}

	authInfos := make(map[string]*api.AuthInfo)
	authInfos["envtest-user"] = &api.AuthInfo{
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
	}

	contexts := make(map[string]*api.Context)
	contexts["envtest-context"] = &api.Context{
		Cluster:   "envtest-cluster",
		AuthInfo:  "envtest-user",
		Namespace: "default", // or whatever namespace you're testing in
	}

	kubeconfig := &api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		Clusters:       clusters,
		AuthInfos:      authInfos,
		Contexts:       contexts,
		CurrentContext: "envtest-context",
	}

	// Create a temporary file to write the kubeconfig to
	kubeconfigPath := filepath.Join(os.TempDir(), "kubeconfig")
	if err := clientcmd.WriteToFile(*kubeconfig, kubeconfigPath); err != nil {
		return "", fmt.Errorf("failed to write kubeconfig file: %w", err)
	}
	return kubeconfigPath, nil
}
