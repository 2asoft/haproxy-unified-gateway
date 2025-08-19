package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/cmd/controller/version"
	ctrlconfig "github.com/haproxytech/kubernetes-controller/controller/configuration"
	haproxymgr "github.com/haproxytech/kubernetes-controller/controller/haproxy"
	haproxyparams "github.com/haproxytech/kubernetes-controller/controller/haproxy/params"
	"github.com/haproxytech/kubernetes-controller/controller/startup"
	controller "github.com/haproxytech/kubernetes-controller/k8s/gate"
	gateconfig "github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	opt "github.com/haproxytech/kubernetes-controller/k8s/gate/options"

	"github.com/joho/godotenv"
	"k8s.io/apimachinery/pkg/types"
)

//revive:disable

// var testKubeConfig = `
// apiVersion: v1
// clusters:
// - cluster:
//     certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJY1ZHaHZlOXRsUk13RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBM01UWXdOekUzTWpKYUZ3MHpOVEEzTVRRd056SXlNakphTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUUROb2hVc09MaHJDV3E0VWtHbVdtelFjeVUvUGtUTmIrRzF3WkRadkZCL3lQZy82RXZXUWF3ZzRDdFYKbTNHS3hSdGE0VmtlYTBPVVRJRzRMYmphbGRKbHFhOERScEcrUG54aVdYTVdMV3d5T2dPMjRmV0ltbUluNXZqZQpvaHd4UTlKYjJnYzM0VmJKTWYrNmxwbEtBUExXS3FlNHlSUmdUL1c0Y3prWVpnM1pKckNad0ZXdkt0MERPeWU2Cis0SGRhN3RoZnhLK0liaXQ5RGx3dWJDU3BkSG0yUnU5Q2Zvcm4zc0p5elp5U29kYVZlSWdyN1Bmc1IvbEF4ZXUKaXl6dER0VjNUaG54dE1CTlBOMWQ3ekg4K1RvK21NQit4WENBSTUrQzJXUWRmaGx6Rko5TEVvOUhYV1FGSU1tKwpLWEdZYzlzZS9kV0Job0hvQ1duQVY3aEQ4cExCQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJUdFRlTDc5VkJIUTBiM3pJK1p4djEvUmM2a1RqQVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQmYvc1VOUVllMwpqNE1QRXJ2bFlmczhrbFNNQmpMYVFZeS80Q29TdUsrQ1k2Y2JpOWVEcEJsNEV4eVVFOGhlMzBaNjUzUlJObHVvClNjUGt5M2hLRVNRUVNYZTE2cnpySjdiYmlTVjVIcnFsSm4vUlBBU1FqL2RLVEcxdjVQeGdXSXExbmwyeDJlR2kKbEtWSXR3SXR4U1FTZVB2Q2hQdzUySlJvVVhPUmVtNHJobUh3QVZveTgwZlBZVlBtcXQ4MDB2czNlTWVFNlh3Wgpqd3ppVVNJRElPQkw5cG1UUTVOSWxTRy9aL3pGUlJGaUlrZW5JaEVVSFllS1V6bjVlYmMrb25uWGltWWg2U0VBCmF4SzhBdTI5U0N2SlEwNjVaWGtnblJpWWVWczU3NW00c1g1ZHVLdS8rOUhnUExBbTYzSWx6U0xteGx1Y2J2Q0EKaVFxTERlNjVsY3gvCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
//     server: https://127.0.0.1:7443
//   name: kind-dev-controller
// contexts:
// - context:
//     cluster: kind-dev-controller
//     user: kind-dev-controller
//   name: kind-dev-controller
// current-context: kind-dev-controller
// kind: Config
// preferences: {}
// users:
// - name: kind-dev-controller
//   user:
//     client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURLVENDQWhHZ0F3SUJBZ0lJZFgwTmJxWkc0aE13RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBM01UWXdOekUzTWpKYUZ3MHlOakEzTVRZd056SXlNakphTUR3eApIekFkQmdOVkJBb1RGbXQxWW1WaFpHMDZZMngxYzNSbGNpMWhaRzFwYm5NeEdUQVhCZ05WQkFNVEVHdDFZbVZ5CmJtVjBaWE10WVdSdGFXNHdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFERTVkeVAKZzZUcXRYSkY0T0lvdkVaN2dMZnF6NFdSbEJncm8xQU90cVM2YnNVMG1uWXlhSG5TQU1JYlNaY0ptRGNYWE1LcQpTSzh0SlBVTHVFR0M0WFpRWXRPdUZCajFBVTU0Ykw2alpQdXU3cFZvU1JsbC9VVzJnVTNHNFVKSEVOMkFXWTFBCi9ZUGQwWTgyZklDMlNSUzlxK0dKaFFtUndDTkt0czVIMWFxb3kyUndEdDFhTHdQMFUzWUF6UUVFSDdsTnRndVIKMEh4NzVkVDBHRldxQjBZVk1YWW5wY3BIeElqclhCQzdIRlhJbXNUcGVVWWpxRWRULy9qNGEvYU5OQlUranVWMQpFYzJWaEtPUUpMdnFCMnhIbWw5amtmTEJ2M2grbFVTTWxpSGV2dTRMb0I0MHJhSWMzNVY2b2JEMUlybktzYUx3CmFYb3VldHNwNHVSV0lEdGpBZ01CQUFHalZqQlVNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUsKQmdnckJnRUZCUWNEQWpBTUJnTlZIUk1CQWY4RUFqQUFNQjhHQTFVZEl3UVlNQmFBRk8xTjR2djFVRWREUnZmTQpqNW5HL1g5RnpxUk9NQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUURFUzhDbDAyNGZ0SUVCcVpKKzRHYmVuaHU5ClE2cU5lRlBSa3dhN3FSdnhhamxUbUFlVzhDVWQyRnh0cWZSOXY5cGkrOWh4Y3dkZ2p4S3hUVzNuUTRNTU45eWEKV2R5UWNtcGM1Tnd0L2pwaVAwTUdJbW5Za0VPd1lHRmk5NXJPMEM4emRUTGd6SHdka1cvSnAzbDgrbWJsK0NVZgpqUmhQK3hVSUpCR1lYZjlCTXE2ZDNuWHZzS2FBRHhIQzlNSU5qNUN2bkRrQ0VXU0RwNGNqSGZXY3pXajR4WTdsCkRhTnRaRzlocE5wVmJzSjFaYTNKbmlCTGVXaHBrd283MUIrd3FvOXM1QTJEaEZPdkhKalNKMy9FK1plbEdFSWsKdEpKM0hzQndvWjE4MGFpWnVheXBxTmdzaTBNSDZxc2J5SDJwZytlY0NnUlJzVldrbVZJRTA3eCtFOXF2Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
//     client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBeE9YY2o0T2s2clZ5UmVEaUtMeEdlNEMzNnMrRmtaUVlLNk5RRHJha3VtN0ZOSnAyCk1taDUwZ0RDRzBtWENaZzNGMXpDcWtpdkxTVDFDN2hCZ3VGMlVHTFRyaFFZOVFGT2VHeStvMlQ3cnU2VmFFa1oKWmYxRnRvRk54dUZDUnhEZGdGbU5RUDJEM2RHUE5ueUF0a2tVdmF2aGlZVUprY0FqU3JiT1I5V3FxTXRrY0E3ZApXaThEOUZOMkFNMEJCQis1VGJZTGtkQjhlK1hVOUJoVnFnZEdGVEYySjZYS1I4U0k2MXdRdXh4VnlKckU2WGxHCkk2aEhVLy80K0d2MmpUUVZQbzdsZFJITmxZU2prQ1M3Nmdkc1I1cGZZNUh5d2I5NGZwVkVqSlloM3I3dUM2QWUKTksyaUhOK1ZlcUd3OVNLNXlyR2k4R2w2TG5yYktlTGtWaUE3WXdJREFRQUJBb0lCQUNnVk9UVFlFbE1ibEFOSQp1QkdkM21WVysxbm9YQ01hT0Y5dDFDYmlwSjgxWEowTVVzS0pSVDlzbXhkT0FGcmFLMkRzcDg1ZGxKZkduY0lBCmhRbWRWMlllOEVQUVlKSkQ3Vk1UcEMyRUtiNWZZSGdGNVk4L0k1bDNNanVwOE1HaDI4MjhyVVpOTmJLSzdqSWoKMzFuOGY2WHJIek5OSzNrSjJjVmtlSkxrR3VWWWd4S05TdG03cVh5NkM0MlRIcmhyRmtyK01MVmNhL3FrcFhGbgpweWtTdFlXeVVJZk9mSjFZa2tka2I0UFlldHdLNnpibEU4dE1pWUdtRm1ReUM4VERqSlgzWlZjZFVnckFkcCtzCmtLOWNXSGZJSFhzUDNiWUtGQW82SXkvMzNZK1hGTThzdUh3c3N6YUsyRGMycERuaXRiZW5Sa0cvbW1zaWRYcDUKelNCQnY2RUNnWUVBM1J3Z2NLaHFENjBjZUlacm53dk9wcy83Q0s3NFREaWRHV0hvZ1JmQ3JpaDU0MEUzUXdVMApIV0dwZ2NsZEFuY2ZGOUxlQTV6emE1bWpDRk9KdlJacm5WRDBLem5TVkE0QllZUzVVNnFFVjV3R3VpTVFDRzF6Ckh4UXdVMWYrR0drMnNQVFJxeEwyS0FBY24xTldZY1FtcnNSQi9adU5SS3lkOHZqQlBXaVJrT0VDZ1lFQTQvZXQKUmRRQVBkNER2aGpTODhmZXZDbGpqcFhOR2MwK3hiaDhGOEV4LzRzRGYxTm14eDhvZzJpa1FWeGdaYk5BWVNlRwpZU0pDWGFOdzNjSGx3L0NMckdOMWZ4THBRa2oxZk81Q1o1Wm55ZXVDcUt4MnAvemNoenMwL2c4K01tKzhEdlNrCkpXdmJSbHRrRWhtZG5sVXZOT0NZMkQ4L1VDZTZmWjhrVTYyNjRNTUNnWUJBM0pSamwvUHMvMUkveE9iak5CcDkKOHJyb1ZET0FZSWN0UC94dGlpUFE1UXpFYm9nZ2YvRkd3VFJ4WHptS2xKa3BhdkUzekIzWUxheVdyN0xUSmpXUgpZNE1NL3h4RkRncTNxYkNYNjRpQkRzTW1iVXl4dkRHdUowVDUzZkVyQmdwR0pMc3czUklhcjlXMW8wUE8wRFNzCnhlTzUycHk1VFkzVURjYmFGY2ZGNFFLQmdCc0hFRm9KQ29aTFBqSlppeGt3Qnk1VDBlUGp5czlXVUN6czlIbDAKaEZNQnprWllRd1Uwb244Qjl3ZHd4bFVJYlllWFFnMWVISFF4bm40TU1RdU1CMk5HMzNWVGJxaFhNaE8vdzh1NApQMUhuUkRSdlRob1lscVRKMWp5UTNoVG92bWtmaEI2VHJRbW9hRExsS3BUTkVLMjZPeVRZU3M5Y0JuWkNXZkk1CjFNQTFBb0dBUy9ka2RiTGFSMGJ0d0l6MDdiUHRjdjNPSW9mTWR3M3FjOFBXeWZ3NWxJK3JJZ1JBb1VsMm5ad1gKWkQ5YXg5dnBWbGJ4N0VIbW5JRCsyVnc4ZTdTZE5PdTBDemhWZE11dkFtMmdDMzcveGN6eWUrSERrWXFHOUR2YQpYVXMxRGxwL1lMemJuZVowNU4vWWtEOUVSUE5qbkV2RTI4NXA0REFOVXYwODVob09CQlk9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
// `

//revive:enable

func main() {
	_ = godotenv.Load()
	fmt.Println(string(version.Info))
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)

	// Controller ctrlConfig from Flags
	ctrlConfig, err := ctrlconfig.Get()
	if err != nil {
		panic(err)
	}

	// GateConfig
	opts := setupGateConfig(ctrlConfig)

	// Start controller
	cntlr, err := controller.New(opts)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	go func() {
		err := cntlr.Run(ctx, &wg)
		if err != nil {
			panic(err)
		}
	}()

	// Start the HAProxy configuration manager
	params := haproxyparams.Params{
		Test:             ctrlConfig.Test,
		UseWiths6Overlay: ctrlConfig.UseWiths6Overlay,
		HaproxyDirs:      ctrlConfig.HaproxyDirs,
	}
	haproxyCfgManager, err := haproxymgr.NewAppManager(ctx, &wg,
		cntlr.Configuration.TransferHaproxyConfChannel,
		params,
		cntlr.Configuration.Logger)
	if err != nil {
		panic(err)
	}
	haproxyCfgManager.Run()

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
	haproxyCfgManager.Stop()

	// Wait for background goroutines to finish
	wg.Wait()
	cntlr.Configuration.Logger.Info("Graceful shutdown complete. Exiting.")
}

func setupGateConfig(ctrlConfig ctrlconfig.ControllerConfig) gateconfig.GateConfigOptions {
	metricsConfig := gateconfig.MetricsConfig{
		Port:    6062,
		Enabled: false,
		Secure:  false,
	}

	// kubeconfig := testKubeConfig
	kubeconfig := ""

	// Optional : default values provided in controller.New()
	// To adjust more precisely the log levels, use the CRD: HaproxyGateCtrlCfg
	// along with opt.ControllerConfCRD to specify which CRD to watcg
	logLevelIfCategoryEmpty := slog.LevelInfo
	logCategoryLevels := map[v3.Category]slog.Level{
		logging.LogCategoryK8s:    slog.LevelInfo,
		logging.LogCategoryGate:   slog.LevelDebug,
		logging.LogCategoryStatus: slog.LevelInfo,
		logging.LogCategoryBatch:  slog.LevelInfo,
	}

	haproxyConfCh := make(chan haproxy.HaproxyConfDiffs, 100)

	// Read the haproy.cfg file at startup, and initializes the library with the initial haproxy configuration
	initialStructured, err := startup.StructuredFromFile(ctrlConfig.HaproxyDirs.MainCfgFile, ctrlConfig.HaproxyDirs.CfgDir)
	if err != nil {
		panic(err)
	}

	opts := gateconfig.GateConfigOptions{
		opt.KubeConfig(kubeconfig),
		opt.ControllerConfCRD(types.NamespacedName{
			Namespace: ctrlConfig.ControllerConfCRD.Namespace,
			Name:      ctrlConfig.ControllerConfCRD.Name,
		}),
		opt.SyncPeriod(ctrlConfig.SyncPeriod),
		opt.StartupSyncPeriod(ctrlConfig.StartupSyncPeriod),
		opt.MetricsConfig(metricsConfig),
		opt.LeaderElectionConfig(ctrlConfig.LeaderElectionEnabled),
		opt.ControllerName(ctrlConfig.ControllerName),
		opt.Namespaces(ctrlConfig.Namespaces),
		opt.Logging(logging.LogHandlerType(ctrlConfig.LogType), logLevelIfCategoryEmpty, logCategoryLevels),
		opt.HaproxyConfChannel(haproxyConfCh),
		opt.IPV4BindAddr(ctrlConfig.IPV4BindAddr),
		opt.IPV6BindAddr(ctrlConfig.IPV6BindAddr),
		opt.HaproxyDirs(ctrlConfig.HaproxyDirs),
		opt.LinkID("link1"),
		opt.InitialStructured(initialStructured),
		opt.CacheReSyncPeriod(ctrlConfig.CacheResyncPeriod),
		opt.DefaultsSectionName(gateconfig.DefaultsSectionName),
	}
	if ctrlConfig.DisableIPv4 {
		opts = append(opts, opt.DisableIPv4())
	}
	if ctrlConfig.DisableIPv6 {
		opts = append(opts, opt.DisableIPv6())
	}
	return opts
}
