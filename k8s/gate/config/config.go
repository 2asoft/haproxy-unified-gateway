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
package config

import (
	"log/slog"
	"os"
	"time"

	v3 "github.com/haproxytech/kubernetes-controller/api/gate/v3"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"k8s.io/apimachinery/pkg/types"
)

// func getConfig() (*rest.Config, error) {
// 	kubeconfig := os.Getenv("PROBER_KUBECONFIG")
// 	if len(kubeconfig) > 0 {
// 		return clientcmd.BuildConfigFromFlags("", kubeconfig)
// 	}

// 	kubeconfig = os.Getenv("KUBECONFIG")
// 	if len(kubeconfig) > 0 {
// 		return clientcmd.BuildConfigFromFlags("", kubeconfig)
// 	}

// 	return config.GetConfig()
// }

type GateConfigOptions []func(c *Configuration) error

type Configuration struct {
	Logger                     *slog.Logger
	LogHandler                 *logging.CategoryFilterHandler
	TransferHaproxyConfChannel chan haproxy.HaproxyCfgDiffs
	// ControllerPodConfig contains information about this Pod.
	ControllerPodConfig ControllerPodConfig
	//  Namespace and name of the controller conf CRD:  HaproxyGateCtrlCfg
	ControllerConfCRD types.NamespacedName
	Kubeconfig        string
	// ControllerName is the name of this controller.
	ControllerName string
	// LindID: an ID for the link to the cluster
	LinkID string
	// HaproxyConfiguration contains the needed configuration to compute the FE/BE/...
	HaproxyConfiguration
	// LeaderElectionConfig contains the configuration for leader election.
	LeaderElectionConfig LeaderElectionConfig
	// WhiteListNamespaces is a list of namespaces to watch.
	// If empty, all namespaces are watched.
	WhiteListNamespaces []string
	// MetricsConfig specifies the metrics config.
	MetricsConfig MetricsConfig
	// SyncPeriod is the duration we wait after handling one batch before the next one
	SyncPeriod time.Duration
}

type HaproxyConfiguration struct {
	// HaproxyDirs contains all the needed dir
	// used for example in map files reference
	HaproxyDirs
	// FrontendNameTemplate: the template for the frontend name.
	FrontendNameTemplate string
	// BackendNameTemplate: the template for the backend name.
	BackendNameTemplate string
	// ServerNameTemplate: the template for the server name.
	ServerNameTemplate string
	// IPv4BindAddress is the IPv4 address to bind to.
	IPv4BindAddress string
	// IPv6BindAddress is the IPv6 address to bind to.
	IPv6BindAddress string
	// DisableIPv4 indicates whether IPv4 is disabled.
	DisableIPv4 bool
	// DisableIPv6 indicates whether IPv6 is disabled.
	DisableIPv6 bool
}

type HaproxyDirs struct {
	CfgDir        string
	MainCfgFile   string
	HaproxyBinary string
	RuntimeDir    string
	StateDir      string
	AuxDir        string
	PIDFile       string
	RuntimeSocket string
	MasterSocket  string
	MapsDir       string
	PatternDir    string
	ErrFileDir    string
}

// ControllerPodConfig contains information about this Pod.
type ControllerPodConfig struct {
	// PodIP is the IP address of this Pod.
	PodIP string
	// Namespace is the namespace of this Pod.
	Namespace string
	// Name is the name of the Pod.
	Name string
}

// MetricsConfig specifies the metrics config.
type MetricsConfig struct {
	// Port is the port the metrics should be exposed on.
	Port int
	// Enabled is the flag for toggling metrics on or off.
	Enabled bool
	// Secure is the flag for toggling the metrics endpoint to https.
	Secure bool
}

// LeaderElectionConfig contains the configuration for leader election.
type LeaderElectionConfig struct {
	// LockName holds the name of the leader election lock.
	LockName string
	// Enabled indicates whether leader election is enabled.
	Enabled bool
}

func NewGateLogger(defaultLevel slog.Level, categoryLevels map[v3.Category]slog.Level) (*slog.Logger, *logging.CategoryFilterHandler) {
	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: defaultLevel,
		// AddSource: true,
	})

	handlerParams := logging.CategoryFilterHandlerParams{
		Base:                  base,
		DefaultLevel:          defaultLevel,
		DefaultCategoryLevels: categoryLevels,
		CategoryKey:           logging.LogCategoryKey,
	}
	handler := logging.NewCategoryFilterHandler(handlerParams)
	slogLogger := slog.New(handler)

	return slogLogger, handler
}
