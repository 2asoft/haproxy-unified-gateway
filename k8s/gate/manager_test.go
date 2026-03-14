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
package controller

import (
	"testing"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/config"
)

func TestGetMetricsOptions_Disabled(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: false,
		Port:    31060,
	}
	opts := getMetricsOptions(cfg)
	if opts.BindAddress != "0" {
		t.Errorf("expected BindAddress '0' when disabled, got %q", opts.BindAddress)
	}
	if opts.SecureServing {
		t.Error("expected SecureServing false when disabled")
	}
	if opts.FilterProvider != nil {
		t.Error("expected nil FilterProvider when disabled")
	}
}

func TestGetMetricsOptions_AuthNone(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled:  true,
		Port:     31060,
		AuthMode: config.MetricsAuthNone,
	}
	opts := getMetricsOptions(cfg)
	if opts.BindAddress != ":31060" {
		t.Errorf("expected BindAddress ':31060', got %q", opts.BindAddress)
	}
	if opts.SecureServing {
		t.Error("expected SecureServing false for auth mode none")
	}
	if opts.FilterProvider != nil {
		t.Error("expected nil FilterProvider for auth mode none")
	}
}

func TestGetMetricsOptions_AuthKubeRBAC(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled:  true,
		Port:     31060,
		AuthMode: config.MetricsAuthKubeRBAC,
	}
	opts := getMetricsOptions(cfg)
	if opts.BindAddress != ":31060" {
		t.Errorf("expected BindAddress ':31060', got %q", opts.BindAddress)
	}
	if !opts.SecureServing {
		t.Error("expected SecureServing true for kube-rbac")
	}
	if opts.FilterProvider == nil {
		t.Error("expected non-nil FilterProvider for kube-rbac")
	}
}

func TestGetMetricsOptions_AuthBasic(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled:           true,
		Port:              31060,
		AuthMode:          config.MetricsAuthBasic,
		BasicAuthUser:     "prometheus",
		BasicAuthPassword: "secret",
	}
	opts := getMetricsOptions(cfg)
	if opts.BindAddress != ":31060" {
		t.Errorf("expected BindAddress ':31060', got %q", opts.BindAddress)
	}
	if !opts.SecureServing {
		t.Error("expected SecureServing true for basic auth")
	}
	if opts.FilterProvider == nil {
		t.Error("expected non-nil FilterProvider for basic auth")
	}
}

func TestGetMetricsOptions_SecureWithoutAuth(t *testing.T) {
	cfg := config.MetricsConfig{
		Enabled: true,
		Port:    31060,
		Secure:  true,
	}
	opts := getMetricsOptions(cfg)
	if !opts.SecureServing {
		t.Error("expected SecureServing true when Secure is set")
	}
	if opts.FilterProvider != nil {
		t.Error("expected nil FilterProvider when no auth mode set")
	}
}
