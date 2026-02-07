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

package httproute

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/haproxytech/haproxy-unified-gateway/test/integration/utils"
)

func (s *HTTPRouteTestSuite) Test_HugConf_DefaultLogFormat() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixturePath := path.Join(fixtureDirPath, "hugconf", "log-format")

	s.CreateFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})

	s.CreateFixtures(fixturePath, []string{"namespace.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"namespace.yaml"})

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf.yaml"})

	expectedLogFormat := "log-format '%[capture.req.hdr(0)] %ci headers=%[var(txn.req_header_names)]'"
	frontendName := fmt.Sprintf("link1_%s_gateway_http", s.Test().Namespace)
	cfgPath := path.Join(s.Test().HaproxyCfgDir, "haproxy.cfg")
	var gotCfg string
	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		gotCfg = string(data)
		if !strings.Contains(gotCfg, frontendName) {
			return false
		}
		if !strings.Contains(gotCfg, expectedLogFormat) {
			return false
		}
		if !strings.Contains(strings.ToLower(gotCfg), "http-request capture req.hdr(authorization) len 256") {
			return false
		}
		return strings.Contains(strings.ToLower(gotCfg), "http-request set-var(txn.req_header_names) req.hdr_names")
	}) {
		s.T().Fatalf("log format or header capture not applied in %s", cfgPath)
	}
	if strings.Contains(gotCfg, "tune.http.logurilen") || strings.Contains(gotCfg, "len 65535") {
		s.T().Fatalf("log line length tuning should be omitted in %s", cfgPath)
	}
}

func (s *HTTPRouteTestSuite) Test_HugConf_LogLineLengthTuning() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixturePath := path.Join(fixtureDirPath, "hugconf", "log-format")

	s.CreateFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})

	s.CreateFixtures(fixturePath, []string{"namespace.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"namespace.yaml"})

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf-log-length.yaml"})

	cfgPath := path.Join(s.Test().HaproxyCfgDir, "haproxy.cfg")
	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		cfg := string(data)
		return strings.Contains(cfg, "log stdout len 65535") &&
			strings.Contains(cfg, "tune.http.logurilen 65535")
	}) {
		s.T().Fatalf("log line length tuning not applied in %s", cfgPath)
	}
}

func (s *HTTPRouteTestSuite) Test_HugConf_LogLineLengthTuning_RemovesUnsetValues() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixturePath := path.Join(fixtureDirPath, "hugconf", "log-format")

	s.CreateFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})

	s.CreateFixtures(fixturePath, []string{"namespace.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"namespace.yaml"})

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf-log-length.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, "test", []string{"hugconf-log-length.yaml"})

	cfgPath := path.Join(s.Test().HaproxyCfgDir, "haproxy.cfg")
	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		cfg := string(data)
		return strings.Contains(cfg, "log stdout len 65535") &&
			strings.Contains(cfg, "tune.http.logurilen 65535")
	}) {
		s.T().Fatalf("log line length tuning not applied in %s", cfgPath)
	}

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf-initial.yaml"})

	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		cfg := string(data)
		if !strings.Contains(cfg, "log stdout format raw daemon") {
			return false
		}
		if strings.Contains(cfg, "log stdout len 65535") {
			return false
		}
		return !strings.Contains(cfg, "tune.http.logurilen")
	}) {
		s.T().Fatalf("log line length tuning not removed from %s", cfgPath)
	}
}

func (s *HTTPRouteTestSuite) Test_HugConf_DefaultLogFormat_UpdatesExistingFrontends() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixturePath := path.Join(fixtureDirPath, "hugconf", "log-format")

	s.CreateFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})

	s.CreateFixtures(fixturePath, []string{"namespace.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"namespace.yaml"})

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf-initial.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, "test", []string{"hugconf-initial.yaml"})

	frontendName := fmt.Sprintf("link1_%s_gateway_http", s.Test().Namespace)
	cfgPath := path.Join(s.Test().HaproxyCfgDir, "haproxy.cfg")
	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		return strings.Contains(string(data), frontendName)
	}) {
		s.T().Fatalf("frontend %s not present in %s", frontendName, cfgPath)
	}

	initialCfg, err := os.ReadFile(cfgPath)
	s.Require().NoError(err)
	if strings.Contains(string(initialCfg), "req_header_names") {
		s.T().Fatal("log format already applied before HugConf update")
	}

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf.yaml"})

	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		cfg := string(data)
		if !strings.Contains(cfg, frontendName) {
			return false
		}
		if !strings.Contains(cfg, "log-format '%[capture.req.hdr(0)] %ci headers=%[var(txn.req_header_names)]'") {
			return false
		}
		if !strings.Contains(strings.ToLower(cfg), "http-request capture req.hdr(authorization) len 256") {
			return false
		}
		return strings.Contains(strings.ToLower(cfg), "http-request set-var(txn.req_header_names) req.hdr_names")
	}) {
		s.T().Fatalf("log format or header capture not applied in %s", cfgPath)
	}
}

func (s *HTTPRouteTestSuite) Test_HugConf_DefaultLogFormat_DisablesRequestHeaderNames() {
	fixtureDirPath := utils.GetCRDFixturePath()
	fixturePath := path.Join(fixtureDirPath, "hugconf", "log-format")

	s.CreateFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"gatewayclass.yaml", "gateway.yaml"})

	s.CreateFixtures(fixturePath, []string{"namespace.yaml"})
	defer s.CleanupFixtures(fixturePath, []string{"namespace.yaml"})

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf.yaml"})
	defer s.CleanupFixturesInNamespace(fixturePath, "test", []string{"hugconf.yaml"})

	cfgPath := path.Join(s.Test().HaproxyCfgDir, "haproxy.cfg")
	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		cfg := string(data)
		if !strings.Contains(cfg, "headers=%[var(txn.req_header_names)]") {
			return false
		}
		return strings.Contains(strings.ToLower(cfg), "http-request set-var(txn.req_header_names) req.hdr_names")
	}) {
		s.T().Fatalf("request header names logging not enabled in %s", cfgPath)
	}

	s.CreateFixturesInNamespace(fixturePath, "test", []string{"hugconf-log-header-names-disabled.yaml"})

	if !utils.WaitFor(s.Test().Ctx, interval, timeout, func() bool {
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return false
		}
		cfg := string(data)
		if !strings.Contains(cfg, "log-format '%[capture.req.hdr(0)] %ci'") {
			return false
		}
		if strings.Contains(cfg, "headers=%[var(txn.req_header_names)]") {
			return false
		}
		return !strings.Contains(strings.ToLower(cfg), "http-request set-var(txn.req_header_names) req.hdr_names")
	}) {
		s.T().Fatalf("request header names logging not disabled in %s", cfgPath)
	}
}
