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
package opt

import (
	"github.com/haproxytech/kubernetes-controller/k8s/gate/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/klog/v2"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	ctlr_zap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func RLogging() func(o *config.Configuration) error {
	return func(o *config.Configuration) error {
		opts := ctlr_zap.Options{
			Development: true,
			TimeEncoder: zapcore.ISO8601TimeEncoder,
			ZapOpts: []zap.Option{
				zap.AddCaller(),
			},
			DestWriter: o.K8sLogging.LogConverter, // use main logger to write output
		}
		logger := ctlr_zap.New(ctlr_zap.UseFlagOptions(&opts))

		runtimelog.SetLogger(logger)
		klog.SetLogger(logger)

		o.K8sLogging.RLogger = logger
		return nil
	}
}
