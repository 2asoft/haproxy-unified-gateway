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
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/haproxytech/client-native/v6/runtime"
	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/cmd/controller/version"
	hugconfig "github.com/haproxytech/kubernetes-controller/hug/configuration"
	haproxymgr "github.com/haproxytech/kubernetes-controller/hug/haproxy"
	haproxyparams "github.com/haproxytech/kubernetes-controller/hug/haproxy/params"
	"github.com/haproxytech/kubernetes-controller/hug/startup"
	controller "github.com/haproxytech/kubernetes-controller/k8s/gate"
	gateconfig "github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/diffs"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy/storage"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	opt "github.com/haproxytech/kubernetes-controller/k8s/gate/options"

	"github.com/joho/godotenv"
	"k8s.io/apimachinery/pkg/types"
)

func main() {
	_ = godotenv.Load()
	fmt.Println(string(version.Info))
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	// Controller HUGConfig from Flags
	hugConfig, err := hugconfig.Get()
	if err != nil {
		panic(err)
	}

	// Setup Gate lib configuration from HUG binary configuration
	opts := setupGateConfig(hugConfig)

	// Start controller
	cntlr, err := controller.New(opts)
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup

	// Setup the HAProxy configuration manager
	var runtimeClientCh chan runtime.Runtime
	if cntlr.Configuration.HaproxyParams.RuntimeUpdateHaproxy {
		// Buffered Channel to not block AppManager
		runtimeClientCh = make(chan runtime.Runtime, 1)
	}

	params := haproxyparams.Params{
		Test:             hugConfig.Test,
		UseWiths6Overlay: hugConfig.UseWiths6Overlay,
		HaproxyDirs:      hugConfig.HaproxyDirs,
	}
	haproxyAppManager, err := haproxymgr.NewAppManager(ctx, &wg,
		cntlr.Configuration.TransferHaproxyConfChannel,
		runtimeClientCh,
		params,
		cntlr.Configuration.Logger)
	if err != nil {
		panic(err)
	}

	// Wait for the runtime client
	if cntlr.Configuration.HaproxyParams.RuntimeUpdateHaproxy {
		runtimeClient, err := waitWithTimeoutForRuntimeClient(cntlr.Configuration.Logger, runtimeClientCh, cntlr.Configuration.HaproxyParams.TimeoutWaitForRuntime)
		if err != nil {
			panic(err)
		}
		cntlr.RuntimeClient = runtimeClient
	}

	// ----------------
	// Start Haproxy App manager
	haproxyAppManager.Run()

	// ----------------
	// Start the controller
	go func() {
		err := cntlr.Run(ctx, &wg)
		if err != nil {
			panic(err)
		}
	}()

	// --------------
	// Shutdown
	// --------------
	// refer to beginning of main
	// 	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
	// ctx.Done() also called on signal received
	<-ctx.Done()
	cntlr.Configuration.Logger.Info("Context cancelled: shutting down controller")
	cntlr.Configuration.Logger.Info("Graceful shutdown requested...")

	// Stop your controller logic
	haproxyAppManager.Stop()

	// Wait for background goroutines to finish
	wg.Wait()
	cntlr.Configuration.Logger.Info("Graceful shutdown complete. Exiting.")
}

func setupGateConfig(hugConfig hugconfig.HUGConfig) gateconfig.GateConfigOptions {
	metricsConfig := gateconfig.MetricsConfig{
		Port:    6062,
		Enabled: false,
		Secure:  false,
	}

	// kubeconfig := testKubeConfig
	kubeconfig := ""

	// Optional : default values provided in controller.New()
	// To adjust more precisely the log levels, use the CRD: HugConf
	// along with opt.ControllerConfCRD to specify which CRD to watcg
	logLevelIfCategoryEmpty := slog.LevelInfo
	logCategoryLevels := map[v3.Category]slog.Level{
		logging.LogCategoryK8s:          slog.LevelInfo,
		logging.LogCategoryGate:         slog.LevelDebug,
		logging.LogCategoryStatus:       slog.LevelInfo,
		logging.LogCategoryBatch:        slog.LevelInfo,
		logging.LogCategoryApp:          slog.LevelInfo,
		logging.LogCategoryCertsStorage: slog.LevelInfo,
	}

	haproxyConfCh := make(chan diffs.HaproxyConfDiffs, 100)

	// Read the haproy.cfg file at startup, and initializes the library with the initial haproxy configuration
	initialStructured, err := startup.StructuredFromFile(
		hugConfig.HaproxyDirs.MainCfgFile,
		hugConfig.HaproxyDirs.CfgDir,
		hugConfig.HaproxyDirs.HaproxyBinary,
	)
	if err != nil {
		panic(err)
	}

	opts := gateconfig.GateConfigOptions{
		opt.KubeConfig(kubeconfig),
		opt.ControllerConfCRD(types.NamespacedName{
			Namespace: hugConfig.ControllerConfCRD.Namespace,
			Name:      hugConfig.ControllerConfCRD.Name,
		}),
		opt.SyncPeriod(hugConfig.SyncPeriod),
		opt.StartupSyncPeriod(hugConfig.StartupSyncPeriod),
		opt.MetricsConfig(metricsConfig),
		opt.LeaderElectionConfig(hugConfig.LeaderElectionEnabled),
		opt.ControllerName(hugConfig.ControllerName),
		opt.Namespaces(hugConfig.Namespaces),
		opt.Logging(logging.LogHandlerType(hugConfig.LogType), logLevelIfCategoryEmpty, logCategoryLevels),
		opt.HaproxyConfChannel(haproxyConfCh),
		opt.IPV4BindAddr(hugConfig.IPV4BindAddr),
		opt.IPV6BindAddr(hugConfig.IPV6BindAddr),
		opt.HaproxyDirs(hugConfig.HaproxyDirs),
		opt.LinkID("link1"),
		opt.InitialStructured(initialStructured),
		opt.CacheReSyncPeriod(hugConfig.CacheResyncPeriod),
		opt.DefaultsSectionName(gateconfig.DefaultsSectionName),
		opt.RuntimeUpdate(gateconfig.DefaultWaitForRuntimeTimeout),   // Send commands through runtime in Gate library
		opt.StoreCertificateOnDisk(storage.StructureTypeCertDefault), // Store the certificates on disk
	}
	if hugConfig.DisableIPv4 {
		opts = append(opts, opt.DisableIPv4())
	}
	if hugConfig.DisableIPv6 {
		opts = append(opts, opt.DisableIPv6())
	}
	return opts
}

func waitWithTimeoutForRuntimeClient(logger *slog.Logger, runtimeClientCh chan runtime.Runtime, timeout time.Duration) (runtime.Runtime, error) {
	var err error
	select {
	case runtimeClient := <-runtimeClientCh:
		logger.LogAttrs(context.Background(), slog.LevelInfo, "Runtime client received from AppManager")
		return runtimeClient, nil
	case <-time.After(timeout):
		err = fmt.Errorf("timeout reached. No Runtime client received from AppManagerin %v ", timeout)
		return nil, err
	}
}
