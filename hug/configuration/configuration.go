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
package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/haproxy"
	"github.com/haproxytech/kubernetes-controller/k8s/gate/logging"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

//revive:disable:line-length-limit
type HUGConfig struct {
	haproxy.HaproxyDirs
	ControllerConfCRD NamespaceNameValue `ff:"          long: haproxyctrlconf-crd,     usage: 'namespace/name of the haproxyctrlconf CRD'"`
	ControllerName    string             `ff:"          long: controller-name,         usage: 'spec.controllerName' GatewayClass selector'"`
	IPV4BindAddr      string             `ff:"          long: ipv4-bind-address,       usage: 'IPv4 address to bind to'"`
	IPV6BindAddr      string             `ff:"          long: ipv6-bind-address,	   usage: 'IPv6 address to bind to'"`
	LogType           string             `ff:"          long: log-type,	      		 usage: 'sets up the log output type (possible values: text, json)"`
	External
	Namespaces            CommaSeparatedValues `ff:"          long: namespaces,                      usage: 'comma separated list of namespaces that controller will monitor'"`
	ControllerPort        int                  `ff:"          long: controller-port,                 usage: 'port to listen on for controller data: prometheus'"`
	SyncPeriod            time.Duration        `ff:"          long: sync-period, default: 0,         usage: 'sets the period at which the controller computes HAProxy configuration file (e.g. 5s, 1m)'"`
	StartupSyncPeriod     time.Duration        `ff:"          long: startup-sync-period, default: 0, usage: 'sets the startup period at which the controller computes HAProxy configuration file (e.g. 5s, 1m)'"`
	CacheResyncPeriod     time.Duration        `ff:"          long: cache-resync-period, default: 0, usage: 'sets the controller-runtime manager cache SyncPeriod. If not set, defaults to controller-runtime defaults (10 hours)'"`
	Help                  bool                 `ff:"          long: help,                            usage: 'help'"`
	Test                  bool                 `ff:"short:t,                                         usage: 'simulate running HAProxy'"`
	LeaderElectionEnabled bool                 `ff:"          long: leader-election-enabled,         usage: 'enable leader election'"`
	UseWiths6Overlay      bool                 `ff:"          long:with-s6-overlay,                  usage: 'use s6 overlay to start/stpop/reload HAProxy'"`
	DisableIPv4           bool                 `ff:"          long: disable-ipv4,                    usage: 'disable IPv4 support'"`
	DisableIPv6           bool                 `ff:"          long: disable-ipv6,			         usage: 'disable IPv6 support'"`
}

//revive:enable:line-length-limit

type External struct {
	CfgDir        string `ff:"                     long: external-config-dir,     usage: 'path to HAProxy configuration directory.'"`
	HaproxyBinary string `ff:"                     long: external-haproxy-binary, usage: 'path to HAProxy binary.'"`
	RuntimeDir    string `ff:"                     long: external-runtime-dir,    usage: 'path to HAProxy runtime directory.'"`
	StateDir      string `ff:"                     long: external-state-dir,      usage: 'path to HAProxy state directory.'"`
	AuxDir        string `ff:"                     long: external-aux-dir,        usage: 'path to HAProxy aux directory.'"`
	External      bool   `ff:"short: e,            long: external,                usage: 'use as external Ingress Controller (out of k8s cluster).'"`
}

// NamespaceNameValue used to automatically distinct namespace/name string
type NamespaceNameValue struct {
	Namespace, Name string
}

func (nv *NamespaceNameValue) String() string {
	if nv == nil {
		return ""
	}
	return fmt.Sprintf("%s/%s", nv.Namespace, nv.Name)
}

func (nv *NamespaceNameValue) Set(s string) error {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return errors.New("invalid format: expected namespace/name")
	}
	nv.Namespace = parts[0]
	nv.Name = parts[1]
	return nil
}

type CommaSeparatedValues []string

func (c *CommaSeparatedValues) String() string {
	return strings.Join(*c, ",")
}

func (c *CommaSeparatedValues) Set(s string) error {
	if s == "" {
		*c = nil
		return nil
	}
	*c = strings.Split(s, ",")
	return nil
}

func Get() (HUGConfig, error) {
	configuration := HUGConfig{}

	external := External{}
	osArgsFF := ff.NewFlagSet("unified-kubernetes-gateway")
	err := osArgsFF.AddStruct(&configuration)
	if err != nil {
		return HUGConfig{}, err
	}

	err = osArgsFF.AddStruct(&external)
	if err != nil {
		return HUGConfig{}, err
	}

	err = ff.Parse(osArgsFF, os.Args[1:], ff.WithEnvVars())
	if err != nil {
		return HUGConfig{}, err
	}
	if configuration.Help {
		fmt.Println(ffhelp.Flags(osArgsFF))
		os.Exit(0) //revive:disable:deep-exit
	}
	// --------------
	// Apply defaults
	// --------------
	configuration.HaproxyDirs = HaproxyDefaults()
	if configuration.ControllerPort == 0 {
		configuration.ControllerPort = defaultControllerPort
	}
	if configuration.LogType == "" {
		configuration.LogType = string(logging.LogHandlerTypeJSON)
	}

	// values for external
	err = configuration.initExternal(external)
	if err != nil {
		return HUGConfig{}, err
	}

	for _, dir := range []string{
		configuration.HaproxyDirs.CfgDir, configuration.HaproxyDirs.RuntimeDir,
		configuration.HaproxyDirs.StateDir, configuration.HaproxyDirs.AuxDir,
	} {
		if dir == "" {
			return HUGConfig{}, fmt.Errorf("failed to init controller config: missing config directories [%s]", dir)
		}
	}

	// Binary and main files
	configuration.MainCfgFile = filepath.Join(configuration.HaproxyDirs.CfgDir, "haproxy.cfg")
	configuration.PIDFile = filepath.Join(configuration.HaproxyDirs.RuntimeDir, "haproxy.pid")
	configuration.RuntimeSocket = filepath.Join(configuration.HaproxyDirs.RuntimeDir, "haproxy-runtime-api.sock")
	configuration.MasterSocket = filepath.Join(configuration.HaproxyDirs.RuntimeDir, "haproxy-master.sock")
	if configuration.Test {
		configuration.HaproxyDirs.HaproxyBinary = "echo"
		configuration.RuntimeSocket = ""
		configuration.MasterSocket = ""
	} else if _, err = os.Stat(configuration.HaproxyDirs.HaproxyBinary); err != nil {
		return HUGConfig{}, err
	}

	// Directories
	configuration.CertsDir = filepath.Join(configuration.HaproxyDirs.CfgDir, config.DefaultCertsDirName)
	configuration.CertListDir = filepath.Join(configuration.HaproxyDirs.CfgDir, config.DefaultCertFilesDirName)
	configuration.MapsDir = filepath.Join(configuration.HaproxyDirs.CfgDir, config.DefaultMapsDirName)
	configuration.PatternDir = filepath.Join(configuration.HaproxyDirs.CfgDir, config.DefaultPattenrDirName)
	configuration.ErrFileDir = filepath.Join(configuration.HaproxyDirs.CfgDir, config.DefaultErrFilesDirName)
	for _, d := range []string{
		configuration.CertsDir,
		configuration.CertListDir,
		configuration.MapsDir,
		configuration.ErrFileDir,
		configuration.HaproxyDirs.StateDir,
		configuration.PatternDir,
	} {
		err = os.MkdirAll(d, 0o755)
		if err != nil {
			return HUGConfig{}, err
		}
	}

	return configuration, nil
}

func (c *HUGConfig) initExternal(external External) error {
	if external.External {
		externalDefaults := externalDefaults()
		if externalDefaults.HaproxyBinary != "" {
			if external.HaproxyBinary == "" {
				external.HaproxyBinary = externalDefaults.HaproxyBinary
			}
			if external.CfgDir == "" {
				external.CfgDir = externalDefaults.CfgDir
			}
			if external.RuntimeDir == "" {
				external.RuntimeDir = externalDefaults.RuntimeDir
			}
			if external.StateDir == "" {
				external.StateDir = externalDefaults.StateDir
			}
			if external.AuxDir == "" {
				external.AuxDir = filepath.Join(external.CfgDir, "aux")
			}
		}
		c.External = external
		c.HaproxyDirs.CfgDir = external.CfgDir
		c.HaproxyDirs.HaproxyBinary = external.HaproxyBinary
		c.HaproxyDirs.RuntimeDir = external.RuntimeDir
		c.HaproxyDirs.StateDir = external.StateDir
		c.HaproxyDirs.AuxDir = external.AuxDir
	}
	for _, dir := range []string{
		c.HaproxyDirs.CfgDir, c.HaproxyDirs.RuntimeDir,
		c.HaproxyDirs.StateDir, c.HaproxyDirs.AuxDir,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
