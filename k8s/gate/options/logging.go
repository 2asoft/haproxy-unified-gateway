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
	"log/slog"

	v3 "github.com/haproxytech/haproxy-unified-gateway/api/gate/v3"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/config"
	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/logging"
)

func Logging(handlerType logging.LogHandlerType, defaultLevel slog.Level, logSettings map[v3.Category]slog.Level) func(o *config.Configuration) error {
	return func(o *config.Configuration) error {
		slogLogger, handler := config.NewBaseLogger(handlerType, defaultLevel, logSettings)
		o.Logger = slogLogger
		o.LogHandler = handler
		o.LogHandlerType = handlerType

		return nil
	}
}
