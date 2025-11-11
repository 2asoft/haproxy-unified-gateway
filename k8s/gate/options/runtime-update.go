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
	"time"

	"github.com/haproxytech/haproxy-unified-gateway/k8s/gate/config"
)

// RuntimeUpdate sets the option to perform runtime commands through the runtime socket
// The timeout specifies the max time to wait for the HUG application to sen the runtime.Runtime
// to the library at start up.
func RuntimeUpdate(timeout time.Duration) func(o *config.Configuration) error {
	return func(o *config.Configuration) error {
		o.HaproxyParams.RuntimeUpdateHaproxy = true
		o.HaproxyParams.TimeoutWaitForRuntime = timeout
		return nil
	}
}
