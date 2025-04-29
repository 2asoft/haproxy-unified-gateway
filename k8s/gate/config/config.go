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
)

type Configuration struct {
	Logger     *slog.Logger
	K8sLogging *K8sLogging
	// ControllerPodConfig contains information about this Pod.
	ControllerPodConfig ControllerPodConfig
	GatewayClass        string
	// GatewayCtlrName is the name of this controller.
	GatewayCtlrName string
	// LeaderElectionConfig contains the configuration for leader election.
	LeaderElectionConfig LeaderElectionConfig
	// MetricsConfig specifies the metrics config.
	MetricsConfig MetricsConfig
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
	// Identity is the unique name of the controller used for identifying the leader.
	Identity string
	// Enabled indicates whether leader election is enabled.
	Enabled bool
}
